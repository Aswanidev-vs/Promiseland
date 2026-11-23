package communities

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

const communitySelect = `
SELECT c.id, c.name, c.slug, c.description, c.icon_url, c.banner_url,
       c.visibility,
       (SELECT COUNT(*) FROM community_memberships m
         WHERE m.community_id = c.id AND m.status = 'active'),
       (SELECT COUNT(*) FROM posts p
         WHERE p.community_id = c.id AND p.deleted_at IS NULL),
       (SELECT m.role FROM community_memberships m
         WHERE m.community_id = c.id AND m.user_id = ? AND m.status = 'active'),
       (EXISTS(SELECT 1 FROM community_memberships m
         WHERE m.community_id = c.id AND m.user_id = ? AND m.status = 'active')
        OR c.owner_id = ?),
       c.created_at, c.owner_id, c.posting_enabled
FROM communities c`

func scanCommunity(rows *sql.Rows, row communityRow, viewerID int64) (communityRow, error) {
	var role sql.NullString
	var isMember int
	err := rows.Scan(
		&row.ID, &row.Name, &row.Slug, &row.Description, &row.IconURL, &row.BannerURL,
		&row.Visibility, &row.MemberCount, &row.PostCount, &role, &isMember,
		&row.CreatedAt, &row.OwnerID, &row.PostingEnabled,
	)
	if err != nil {
		return row, err
	}
	row.IsMember = isMember == 1
	if role.Valid && row.IsMember {
		r := role.String
		row.Role = &r
	} else if row.OwnerID == viewerID {
		r := "owner"
		row.Role = &r
		row.IsMember = true
	}
	return row, nil
}

func communityBySlug(ctx context.Context, d *shared.Deps, slug string, viewerID int64) (communityRow, error) {
	var row communityRow
	rows, err := d.DB.QueryContext(ctx, communitySelect+` WHERE c.slug = ?`, viewerID, viewerID, viewerID, slug)
	if err != nil {
		return row, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return row, err
		}
		return row, sql.ErrNoRows
	}
	return scanCommunity(rows, row, viewerID)
}

func communityByID(ctx context.Context, d *shared.Deps, id, viewerID int64) (communityRow, error) {
	var row communityRow
	rows, err := d.DB.QueryContext(ctx, communitySelect+` WHERE c.id = ?`, viewerID, viewerID, viewerID, id)
	if err != nil {
		return row, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return row, err
		}
		return row, sql.ErrNoRows
	}
	return scanCommunity(rows, row, viewerID)
}

// RoleFor returns the caller's effective role in a community: "owner",
// "moderator", "member", or "" when the user is not an active member.
// Posts imports communities only for this helper.
func RoleFor(d *shared.Deps, communityID, userID int64) string {
	if d == nil || d.DB == nil || userID <= 0 || communityID <= 0 {
		return ""
	}
	var ownerID int64
	if err := d.DB.QueryRow(`SELECT owner_id FROM communities WHERE id = ?`, communityID).Scan(&ownerID); err != nil {
		return ""
	}
	if ownerID == userID {
		return "owner"
	}
	var role string
	err := d.DB.QueryRow(`
		SELECT role FROM community_memberships
		WHERE community_id = ? AND user_id = ? AND status = 'active'`,
		communityID, userID).Scan(&role)
	if err != nil {
		return ""
	}
	return role
}

func loadUsers(ctx context.Context, d *shared.Deps, ids []int64) map[int64]UserSummary {
	out := make(map[int64]UserSummary, len(ids))
	if len(ids) == 0 {
		return out
	}
	usernameExpr := "NULLIF(u.username, '')"
	if !columnExists(d, "users", "username") {
		// R0 foundation predates the username column; display name is the
		// stable fallback until the users agent's migration is present.
		usernameExpr = "NULLIF(u.display_name, '')"
	}
	q := fmt.Sprintf(`
		SELECT u.id, COALESCE(%s, u.display_name, ''), u.display_name, p.avatar_url
		FROM users u LEFT JOIN user_profiles p ON p.user_id = u.id
		WHERE u.id IN (%s)`, usernameExpr, placeholders(len(ids)))
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := d.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var u UserSummary
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL); err != nil {
			continue
		}
		out[u.ID] = u
	}
	return out
}

func loadAnime(ctx context.Context, d *shared.Deps, ids []int64) map[int64]Anime {
	out := make(map[int64]Anime, len(ids))
	if len(ids) == 0 || !tableExists(d, "anime_titles") {
		return out
	}
	external := "''"
	if columnExists(d, "anime_titles", "external_id") {
		external = "COALESCE(CAST(a.external_id AS TEXT), '')"
	}
	title := "''"
	if columnExists(d, "anime_titles", "title") {
		title = "COALESCE(a.title, '')"
	}
	alternate := "NULL"
	if columnExists(d, "anime_titles", "alternate_title") {
		alternate = "a.alternate_title"
	}
	format := "''"
	if columnExists(d, "anime_titles", "format") {
		format = "COALESCE(a.format, '')"
	} else if columnExists(d, "anime_titles", "media_type") {
		format = "COALESCE(a.media_type, '')"
	}
	q := fmt.Sprintf(`SELECT a.id, %s, %s, %s, %s FROM anime_titles a WHERE a.id IN (%s)`,
		external, title, alternate, format, placeholders(len(ids)))
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := d.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var a Anime
		var alt sql.NullString
		if err := rows.Scan(&a.ID, &a.ExternalID, &a.Title, &alt, &a.Format); err != nil {
			continue
		}
		a.AlternateTitle = nullString(alt)
		out[a.ID] = a
	}
	return out
}

type postRow struct {
	ID              int64
	AuthorID        int64
	CommunityID     sql.NullInt64
	Type            string
	Title           string
	Body            string
	LinkURL         string
	CreatorCategory string
	Spoiler         bool
	SpoilerNote     string
	Season          int
	Episode         int
	VoteScore       int
	CreatedAt       string
	UpdatedAt       string
	EditedAt        string
}

const postSelect = `
SELECT p.id, p.author_id, p.community_id, p.type, p.title, p.body,
       COALESCE(p.link_url, ''), p.spoiler, COALESCE(p.spoiler_note, ''),
       COALESCE(p.creator_category, ''), COALESCE(p.season, 0), COALESCE(p.episode, 0), p.vote_score,
       p.created_at, COALESCE(p.updated_at, p.created_at), COALESCE(p.edited_at, ''),
       COALESCE(p.is_locked, 0)
FROM posts p`

func scanPostRows(rows *sql.Rows) ([]postRow, error) {
	var out []postRow
	for rows.Next() {
		var p postRow
		var spoiler, locked int
		err := rows.Scan(
			&p.ID, &p.AuthorID, &p.CommunityID, &p.Type, &p.Title, &p.Body,
			&p.LinkURL, &spoiler, &p.SpoilerNote, &p.CreatorCategory, &p.Season, &p.Episode,
			&p.VoteScore, &p.CreatedAt, &p.UpdatedAt, &p.EditedAt, &locked,
		)
		if err != nil {
			return nil, err
		}
		p.Spoiler = spoiler == 1
		out = append(out, p)
	}
	return out, rows.Err()
}

// buildPosts expands raw post rows into the exact R1 post JSON shape.
func buildPosts(ctx context.Context, d *shared.Deps, rows []postRow, viewerID int64) ([]Post, error) {
	out := make([]Post, 0, len(rows))
	authorIDs := make([]int64, 0, len(rows))
	for _, p := range rows {
		authorIDs = append(authorIDs, p.AuthorID)
	}
	authors := loadUsers(ctx, d, authorIDs)

	for _, p := range rows {
		item := Post{
			ID: p.ID, Type: p.Type, Title: p.Title, Body: p.Body,
			LinkURL: p.LinkURL, CreatorCategory: p.CreatorCategory, Spoiler: p.Spoiler, SpoilerNote: p.SpoilerNote,
			Season: p.Season, Episode: p.Episode, VoteScore: p.VoteScore,
			CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt, EditedAt: p.EditedAt,
			Author: authors[p.AuthorID], Media: []Media{}, Anime: []Anime{}, Tags: []string{},
		}
		if item.Author.ID == 0 {
			item.Author = UserSummary{ID: p.AuthorID, DisplayName: "unknown"}
		}
		if p.CommunityID.Valid {
			community, err := communityByID(ctx, d, p.CommunityID.Int64, viewerID)
			if err == nil {
				c := community.Community
				item.Community = &c
			}
		}
		if viewerID > 0 {
			_ = d.DB.QueryRowContext(ctx, `SELECT value FROM votes WHERE user_id = ? AND post_id = ?`,
				viewerID, p.ID).Scan(&item.MyVote)
			var bookmarked int
			_ = d.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM bookmarks WHERE user_id = ? AND post_id = ?)`,
				viewerID, p.ID).Scan(&bookmarked)
			item.Bookmarked = bookmarked == 1
		}
		_ = d.DB.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM comments WHERE post_id = ? AND deleted_at IS NULL`,
			p.ID).Scan(&item.CommentCount)

		mediaRows, err := d.DB.QueryContext(ctx, `
			SELECT m.id, m.storage_key, m.mime_type, m.size_bytes, m.width, m.height
			FROM post_media pm JOIN media m ON m.id = pm.media_id
			WHERE pm.post_id = ? ORDER BY m.id`, p.ID)
		if err == nil {
			for mediaRows.Next() {
				var m Media
				var key string
				if err := mediaRows.Scan(&m.ID, &key, &m.MimeType, &m.SizeBytes, &m.Width, &m.Height); err == nil {
					m.URL = "/media/" + key
					item.Media = append(item.Media, m)
				}
			}
			mediaRows.Close()
		}

		tagRows, err := d.DB.QueryContext(ctx, `SELECT tag FROM post_tags WHERE post_id = ? ORDER BY tag`, p.ID)
		if err == nil {
			for tagRows.Next() {
				var tag string
				if err := tagRows.Scan(&tag); err == nil {
					item.Tags = append(item.Tags, tag)
				}
			}
			tagRows.Close()
		}

		animeIDs := []int64{}
		animeRows, err := d.DB.QueryContext(ctx, `SELECT anime_title_id FROM post_anime WHERE post_id = ? ORDER BY anime_title_id`, p.ID)
		if err == nil {
			for animeRows.Next() {
				var id int64
				if err := animeRows.Scan(&id); err == nil {
					animeIDs = append(animeIDs, id)
				}
			}
			animeRows.Close()
		}
		anime := loadAnime(ctx, d, animeIDs)
		for _, id := range animeIDs {
			if a, ok := anime[id]; ok {
				item.Anime = append(item.Anime, a)
			}
		}

		if p.Type == "poll" {
			var question string
			_ = d.DB.QueryRowContext(ctx, `SELECT COALESCE(poll_question, title) FROM posts WHERE id = ?`, p.ID).Scan(&question)
			if question == "" {
				question = p.Title
			}
			poll := &Poll{Question: question, Options: []PollOption{}}
			optionRows, err := d.DB.QueryContext(ctx, `
				SELECT o.id, o.text,
				       (SELECT COUNT(*) FROM poll_votes v WHERE v.option_id = o.id)
				FROM poll_options o WHERE o.post_id = ? ORDER BY o.sort_order, o.id`, p.ID)
			if err == nil {
				for optionRows.Next() {
					var o PollOption
					if err := optionRows.Scan(&o.ID, &o.Text, &o.Votes); err == nil {
						poll.Options = append(poll.Options, o)
						poll.TotalVotes += o.Votes
					}
				}
				optionRows.Close()
			}
			if viewerID > 0 {
				var optionID int64
				if err := d.DB.QueryRowContext(ctx,
					`SELECT option_id FROM poll_votes WHERE post_id = ? AND user_id = ?`,
					p.ID, viewerID).Scan(&optionID); err == nil {
					poll.MyVote = &optionID
				}
			}
			item.Poll = poll
		}
		out = append(out, item)
	}
	return out, nil
}
