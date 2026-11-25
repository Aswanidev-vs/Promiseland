package posts

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	// See load.go: communities.RoleFor is the only sanctioned cross-module import.
	"github.com/Aswanidev-vs/promiseland/backend/internal/communities"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

func validatePatch(p postPatch) map[string]string {
	details := map[string]string{}
	if p.Type != nil && !allowedTypes[*p.Type] {
		details["type"] = "Choose a valid post type."
	}
	if p.Title != nil {
		n := utf8.RuneCountInString(strings.TrimSpace(*p.Title))
		if n == 0 || n > 200 {
			details["title"] = "Title must be between 1 and 200 characters."
		}
	}
	if p.Body != nil && utf8.RuneCountInString(*p.Body) > 10000 {
		details["body"] = "Body must be at most 10000 characters."
	}
	if p.SpoilerNote != nil && utf8.RuneCountInString(*p.SpoilerNote) > 200 {
		details["spoilerNote"] = "Spoiler note must be at most 200 characters."
	}
	if p.LinkURL != nil && !shared.ValidOptionalURL(*p.LinkURL) {
		details["linkUrl"] = "Links must be an http or https address."
	}
	if p.MediaIDs != nil && len(dedupeIDs(*p.MediaIDs)) > 4 {
		details["mediaIds"] = "A post can have at most 4 media attachments."
	}
	if p.AnimeIDs != nil && len(dedupeIDs(*p.AnimeIDs)) > 5 {
		details["animeIds"] = "A post can reference at most 5 anime titles."
	}
	if p.Tags != nil && len(cleanTags(*p.Tags)) > 5 {
		details["tags"] = "A post can have at most 5 tags."
	}
	if p.Season != nil && *p.Season < 0 || p.Episode != nil && *p.Episode < 0 {
		details["episode"] = "Season and episode must not be negative."
	}
	if p.CreatorCategory != nil && !validCreatorCategory(*p.CreatorCategory) {
		details["creatorCategory"] = "Choose a valid creator category."
	}
	if p.Poll != nil {
		if len(p.Poll.Options) < 2 || len(p.Poll.Options) > 6 {
			details["poll"] = "Polls need between 2 and 6 options."
		} else {
			for _, option := range p.Poll.Options {
				if strings.TrimSpace(option.Text) == "" || utf8.RuneCountInString(option.Text) > 200 {
					details["poll"] = "Each poll option needs text of at most 200 characters."
					break
				}
			}
		}
	}
	return details
}

func (h *handler) patch(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	viewer, _ := shared.UserID(r)
	authorID, communityID, locked, deleted, err := h.rawPost(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) || deleted {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Post not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the post.")
		return
	}
	moderator := h.mayModerate(communityID, authorID, viewer)
	if !moderator {
		shared.WriteError(w, http.StatusForbidden, codeForbidden, "Only the author or community staff can edit this post.")
		return
	}
	if locked && authorID == viewer && !isPlatformModerator(h.d, viewer) &&
		(communityID.Valid && !isCommunityStaff(h.d, communityID.Int64, viewer)) {
		shared.WriteError(w, http.StatusForbidden, codePostLocked, "This post is locked.")
		return
	}
	var p postPatch
	if !shared.DecodeJSON(w, r, &p) {
		return
	}
	if p.IsLocked != nil && authorID == viewer && !isCommunityStaff(h.d, communityID.Int64, viewer) && !isPlatformModerator(h.d, viewer) {
		shared.WriteError(w, http.StatusForbidden, codeForbidden, "Only moderators can lock posts.")
		return
	}
	if details := validatePatch(p); len(details) > 0 {
		writeValidation(w, details)
		return
	}
	if p.MediaIDs != nil && !h.mediaBelongsToAuthor(dedupeIDs(*p.MediaIDs), viewer) {
		writeValidation(w, map[string]string{"mediaIds": "One or more media attachments do not belong to you."})
		return
	}

	sets := []string{"updated_at = datetime('now')", "edited_at = datetime('now')"}
	args := []any{}
	add := func(column string, value any) {
		sets = append(sets, column+" = ?")
		args = append(args, value)
	}
	if p.Type != nil {
		add("type", *p.Type)
	}
	if p.Title != nil {
		add("title", strings.TrimSpace(*p.Title))
	}
	if p.Body != nil {
		add("body", *p.Body)
	}
	if p.LinkURL != nil {
		add("link_url", nullableString(*p.LinkURL))
	}
	if p.Spoiler != nil {
		add("spoiler", boolInt(*p.Spoiler))
	}
	if p.SpoilerNote != nil {
		add("spoiler_note", nullableString(*p.SpoilerNote))
	}
	if p.Poll != nil {
		add("poll_question", nullableString(p.Poll.Question))
	}
	if p.Season != nil {
		add("season", *p.Season)
	}
	if p.Episode != nil {
		add("episode", *p.Episode)
	}
	if p.CreatorCategory != nil {
		add("creator_category", strings.ToLower(strings.TrimSpace(*p.CreatorCategory)))
	}
	if p.IsLocked != nil {
		add("is_locked", boolInt(*p.IsLocked))
	}

	tx, err := h.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not update the post.")
		return
	}
	defer func() { _ = tx.Rollback() }()
	args = append(args, id)
	if _, err := tx.Exec(`UPDATE posts SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not update the post.")
		return
	}
	if p.MediaIDs != nil {
		if _, err := tx.Exec(`DELETE FROM post_media WHERE post_id = ?`, id); err != nil {
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not update media.")
			return
		}
		in := postInput{MediaIDs: *p.MediaIDs}
		if err := h.insertRelations(tx, id, in); err != nil {
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not update media.")
			return
		}
	}
	if p.AnimeIDs != nil {
		if _, err := tx.Exec(`DELETE FROM post_anime WHERE post_id = ?`, id); err != nil {
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not update anime references.")
			return
		}
		in := postInput{AnimeIDs: *p.AnimeIDs}
		if err := h.insertRelations(tx, id, in); err != nil {
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not update anime references.")
			return
		}
	}
	if p.Tags != nil {
		if _, err := tx.Exec(`DELETE FROM post_tags WHERE post_id = ?`, id); err != nil {
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not update tags.")
			return
		}
		in := postInput{Tags: *p.Tags}
		if err := h.insertRelations(tx, id, in); err != nil {
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not update tags.")
			return
		}
	}
	if p.Poll != nil {
		if _, err := tx.Exec(`DELETE FROM poll_votes WHERE post_id = ?`, id); err != nil {
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not update the poll.")
			return
		}
		if _, err := tx.Exec(`DELETE FROM poll_options WHERE post_id = ?`, id); err != nil {
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not update the poll.")
			return
		}
		for i, option := range p.Poll.Options {
			if _, err := tx.Exec(`INSERT INTO poll_options(post_id, text, sort_order) VALUES(?,?,?)`,
				id, strings.TrimSpace(option.Text), i); err != nil {
				shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not update the poll.")
				return
			}
		}
	}
	if err := tx.Commit(); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not update the post.")
		return
	}
	post, err := h.loadPost(r.Context(), id, viewer)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the post.")
		return
	}
	if p.Body != nil {
		h.notifyMentions(viewer, *p.Body, "post", id)
	}
	shared.WriteJSON(w, http.StatusOK, post)
}

func isCommunityStaff(d *shared.Deps, communityID, userID int64) bool {
	if communityID <= 0 {
		return false
	}
	role := communities.RoleFor(d, communityID, userID)
	return role == "owner" || role == "moderator"
}

func (h *handler) remove(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	viewer, _ := shared.UserID(r)
	authorID, communityID, _, deleted, err := h.rawPost(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) || deleted {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Post not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the post.")
		return
	}
	if !h.mayModerate(communityID, authorID, viewer) {
		shared.WriteError(w, http.StatusForbidden, codeForbidden, "Only the author or community staff can remove this post.")
		return
	}
	// Soft delete: comments and votes remain in place to keep threads intact.
	_, err = h.d.DB.ExecContext(r.Context(),
		`UPDATE posts SET deleted_at = datetime('now'), updated_at = datetime('now') WHERE id = ?`, id)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not remove the post.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, nil)
}
