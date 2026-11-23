// Package feed serves the R1 home feed. It is public: anonymous callers see
// public posts only, while signed-in callers get membership/follow/interest
// based results.
package feed

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/feed/shape"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

const errValidation = "validation_error"

// creatorCategories mirrors the R2 vocabulary so GET /feed can filter the
// social feed by creator category without importing another domain module.
var creatorCategories = map[string]bool{
	"cosplay": true, "crossplay": true, "fan_art": true,
	"photography": true, "edit": true, "collection": true, "other": true,
}

type module struct {
	d      *shared.Deps
	loader *shape.Loader
	schema *shape.Schema
}

// Register mounts the public feed routes on r.
func Register(r chi.Router, d *shared.Deps) {
	m := newModule(context.Background(), d)
	if r == nil {
		return
	}
	r.Get("/feed", m.handleFeed)
}

// RegisterProtected is provided for orchestrators that prefer mounting the
// feed inside the authenticated group. The handler itself accepts anonymous
// callers, so both wirings behave identically.
func RegisterProtected(r chi.Router, d *shared.Deps) {
	Register(r, d)
}

func newModule(ctx context.Context, d *shared.Deps) *module {
	s := shape.LoadSchema(ctx, nilIfNilDB(d))
	return &module{
		d:      d,
		schema: s,
		loader: &shape.Loader{DB: nilIfNilDB(d), S: s},
	}
}

func nilIfNilDB(d *shared.Deps) *sql.DB {
	if d == nil {
		return nil
	}
	return d.DB
}

func (m *module) handleFeed(w http.ResponseWriter, r *http.Request) {
	if m == nil || m.d == nil || m.d.DB == nil {
		shared.WriteError(w, http.StatusServiceUnavailable, shared.CodeDBUnavailable, "The feed is temporarily unavailable.")
		return
	}
	q := r.URL.Query()
	scope := strings.TrimSpace(q.Get("scope"))
	if scope == "" {
		scope = "for-you"
	}
	if scope != "for-you" && scope != "following" {
		shared.WriteErrorDetails(w, http.StatusBadRequest, errValidation,
			"Invalid feed scope.", map[string]string{"scope": "must be for-you or following"})
		return
	}
	pg := shared.ParsePagination(r)

	var afterID int64
	if pg.Cursor != "" {
		id, err := shared.DecodeIDCursor(pg.Cursor)
		if err != nil {
			shared.WriteErrorDetails(w, http.StatusBadRequest, errValidation,
				"Invalid pagination cursor.", map[string]string{"cursor": "must be a valid cursor"})
			return
		}
		afterID = id
	}

	viewerID, _ := m.d.SessionUserID(r) // anonymous requests are supported
	where, args := m.where(scope, viewerID, afterID)
	rawCategory := q.Get("creatorCategory")
	if strings.TrimSpace(rawCategory) == "" {
		rawCategory = q.Get("creator_category")
	}
	if strings.TrimSpace(rawCategory) != "" {
		category := strings.ToLower(strings.TrimSpace(rawCategory))
		if !creatorCategories[category] || !m.schema.HasCol("posts", "creator_category") {
			shared.WriteErrorDetails(w, http.StatusBadRequest, errValidation,
				"Invalid creator category.", map[string]string{"creatorCategory": "Choose a valid creator category."})
			return
		}
		where += " AND p.creator_category = ?"
		args = append(args, category)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	posts, err := m.loader.Load(ctx, where, args, viewerID, pg.Limit+1)
	if err != nil {
		if m.d.Log != nil {
			m.d.Log.Warn("feed query failed", "error", err, "scope", scope)
		}
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "The feed could not be loaded.")
		return
	}

	page := shared.NewPage(posts)
	if len(posts) > pg.Limit {
		page.Items = posts[:pg.Limit]
		page.NextCursor = shared.EncodeIDCursor(posts[pg.Limit-1].ID)
	}
	shared.WriteJSON(w, http.StatusOK, page)
}

// where builds a trusted SQL fragment (no user-supplied strings are formatted
// into it) and its bound arguments.
func (m *module) where(scope string, viewerID, afterID int64) (string, []any) {
	s := m.schema
	var clauses []string
	var args []any
	add := func(fragment string, values ...any) {
		clauses = append(clauses, fragment)
		args = append(args, values...)
	}

	if afterID > 0 {
		add("AND p.id < ?", afterID)
	}

	// Private-community posts, blocked authors and suspended authors are
	// excluded from every feed surface.
	visibility, visibilityArgs := shape.VisibilityClauses(s, viewerID)
	if visibility != "" {
		add("AND " + visibility)
		args = append(args, visibilityArgs...)
	}

	viewer := strconv.FormatInt(viewerID, 10)
	memberClause := "0"
	membership := s.MembershipTable()
	if membership != "" && s.HasCol(membership, "user_id") {
		memberClause = fmt.Sprintf(`EXISTS (
			SELECT 1 FROM %s cm
			WHERE cm.community_id = p.community_id AND cm.user_id = %s AND %s
		)`, membership, viewer, s.ActiveMembership("cm"))
	}
	if s.HasCol("communities", "owner_id") {
		memberClause = fmt.Sprintf("(%s OR c.owner_id = %s)", memberClause, viewer)
	}
	followClause := "0"
	if s.HasTable("follows") {
		follower := s.Col("follows", "follower_id", "follower")
		followed := s.Col("follows", "followed_id", "following_id", "followee_id")
		if follower != "" && followed != "" {
			followClause = fmt.Sprintf(`EXISTS (
				SELECT 1 FROM follows f WHERE f.%s = %s AND f.%s = p.author_id
			)`, follower, viewer, followed)
		}
	}

	if viewerID > 0 {
		if scope == "following" {
			add("AND " + followClause)
		} else {
			interestClause := m.interestClause(viewerID)
			add(fmt.Sprintf("AND (%s OR %s OR %s)", memberClause, followClause, interestClause))
		}
	}

	return strings.Join(clauses, " "), args
}

func (m *module) interestClause(viewerID int64) string {
	s := m.schema
	viewer := strconv.FormatInt(viewerID, 10)
	if s.HasTable("post_tags") && s.HasTable("user_interests") {
		postCol := s.Col("post_tags", "post_id")
		if postCol != "" {
			if interestCol := s.Col("post_tags", "interest_id"); interestCol != "" &&
				s.HasCol("user_interests", "interest_id") {
				return fmt.Sprintf(`EXISTS (
					SELECT 1 FROM post_tags pt
					JOIN user_interests ui ON ui.interest_id = pt.%s
					WHERE pt.%s = p.id AND ui.user_id = %s
				)`, interestCol, postCol, viewer)
			}
			tagCol := s.Col("post_tags", "tag", "tag_name", "name")
			if tagCol != "" && s.HasTable("interests") && s.HasCol("interests", "name") &&
				s.HasCol("user_interests", "interest_id") {
				return fmt.Sprintf(`EXISTS (
					SELECT 1 FROM post_tags pt
					JOIN interests i ON i.name = pt.%s
					JOIN user_interests ui ON ui.interest_id = i.id
					WHERE pt.%s = p.id AND ui.user_id = %s
				)`, tagCol, postCol, viewer)
			}
		}
	}
	postAnimeCol := s.Col("post_anime", "anime_title_id", "anime_id")
	userAnimeTable := ""
	for _, candidate := range []string{"anime_interests", "user_anime_interests"} {
		if s.HasTable(candidate) {
			userAnimeTable = candidate
			break
		}
	}
	userAnimeCol := s.Col(userAnimeTable, "anime_title_id", "anime_id")
	if postAnimeCol != "" && userAnimeCol != "" &&
		s.HasCol("post_anime", "post_id") && s.HasCol(userAnimeTable, "user_id") {
		return fmt.Sprintf(`EXISTS (
			SELECT 1 FROM post_anime pa
			JOIN %s uai ON uai.%s = pa.%s
			WHERE pa.post_id = p.id AND uai.user_id = %s
		)`, userAnimeTable, userAnimeCol, postAnimeCol, viewer)
	}
	return "0"
}
