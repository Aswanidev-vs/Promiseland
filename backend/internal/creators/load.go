package creators

import (
	"context"
	"sort"
	"strings"

	"github.com/Aswanidev-vs/promiseland/backend/internal/feed/shape"
)

// The schema is inspected once at registration time. These helpers keep all
// SQL defensive against databases created before a given module's migration.
func (s *service) usernameExpr() string {
	if s.schema.HasCol("users", "username") {
		return "COALESCE(NULLIF(u.username, ''), u.display_name, '')"
	}
	return "COALESCE(u.display_name, '')"
}

func (s *service) displayExpr() string {
	if s.schema.HasCol("users", "display_name") {
		return "COALESCE(u.display_name, '')"
	}
	if s.schema.HasCol("user_profiles", "display_name") {
		return "COALESCE(up.display_name, '')"
	}
	return "''"
}

func (s *service) postAlive(alias string) string {
	if s.schema.HasCol("posts", "deleted_at") {
		return alias + ".deleted_at IS NULL"
	}
	return "1 = 1"
}

func (s *service) buildCreators(ctx context.Context, users []UserSummary, viewerID int64) ([]Creator, error) {
	out := make([]Creator, 0, len(users))
	for _, user := range users {
		creator := Creator{
			User:          user,
			Categories:    []string{},
			FeaturedPosts: []shape.Post{},
		}

		if s.schema.HasTable("posts") && s.schema.HasCol("posts", "creator_category") {
			categoryRows, err := s.d.DB.QueryContext(ctx, `
				SELECT DISTINCT COALESCE(p.creator_category, '')
				FROM posts p WHERE p.author_id = ? AND `+s.postAlive("p")+`
				  AND COALESCE(p.creator_category, '') <> ''
				ORDER BY p.creator_category`, user.ID)
			if err != nil {
				return nil, err
			}
			for categoryRows.Next() {
				var category string
				if err := categoryRows.Scan(&category); err == nil && category != "" {
					creator.Categories = append(creator.Categories, category)
				}
			}
			categoryRows.Close()

			if err := s.d.DB.QueryRowContext(ctx, `
				SELECT COUNT(*) FROM posts p
				WHERE p.author_id = ? AND `+s.postAlive("p")+`
				  AND COALESCE(p.creator_category, '') <> ''`, user.ID).Scan(&creator.PostCount); err != nil {
				return nil, err
			}
		}

		if s.schema.HasTable("follows") {
			_ = s.d.DB.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM follows WHERE followed_id = ?`, user.ID).Scan(&creator.FollowerCount)
		}

		featured, err := s.loadFeatured(ctx, user.ID, viewerID)
		if err != nil {
			return nil, err
		}
		creator.FeaturedPosts = featured
		out = append(out, creator)
	}
	return out, nil
}

type featuredRow struct {
	postID    int64
	sortOrder int
}

// loadFeatured returns the user's curated posts in their chosen order, using
// the shared Post loader so featured posts carry the exact same JSON (and
// visibility filtering) as the home feed.
func (s *service) loadFeatured(ctx context.Context, userID, viewerID int64) ([]shape.Post, error) {
	out := []shape.Post{}
	if !s.schema.HasTable("featured_posts") || !s.schema.HasTable("posts") {
		return out, nil
	}
	rows, err := s.d.DB.QueryContext(ctx, `
		SELECT post_id, sort_order FROM featured_posts
		WHERE user_id = ? ORDER BY sort_order, post_id LIMIT ?`, userID, maxFeatured)
	if err != nil {
		return nil, err
	}
	selected := []featuredRow{}
	for rows.Next() {
		var row featuredRow
		if err := rows.Scan(&row.postID, &row.sortOrder); err == nil {
			selected = append(selected, row)
		}
	}
	rows.Close()
	if len(selected) == 0 {
		return out, nil
	}

	ids := make([]any, 0, len(selected))
	order := make(map[int64]int, len(selected))
	for _, row := range selected {
		ids = append(ids, row.postID)
		order[row.postID] = row.sortOrder
	}

	where := ` AND p.id IN (` + placeholders(len(ids)) + `)`
	args := append([]any{}, ids...)
	visibility, visibilityArgs := shape.VisibilityClauses(s.schema, viewerID)
	if visibility != "" {
		where += " AND " + visibility
		args = append(args, visibilityArgs...)
	}
	posts, err := s.loader.Load(ctx, where, args, viewerID, len(selected))
	if err != nil {
		return nil, err
	}
	sort.SliceStable(posts, func(i, j int) bool {
		if order[posts[i].ID] == order[posts[j].ID] {
			return posts[i].ID < posts[j].ID
		}
		return order[posts[i].ID] < order[posts[j].ID]
	})
	return posts, nil
}

// normalizeList is a small helper kept here to make future filter additions
// easier to reason about without touching handler SQL strings.
func normalizeList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
