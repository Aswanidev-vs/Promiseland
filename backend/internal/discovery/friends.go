package discovery

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

type candidateRow struct {
	user UserProfile
}

// loadCandidates performs the one bounded candidate scan, excluding self,
// inactive/private/non-discoverable accounts, either-direction blocks,
// suppressed candidates and users already connected by an accepted request.
func (m *module) loadCandidates(ctx context.Context, viewerID int64) ([]candidateRow, error) {
	rows, err := m.d.DB.QueryContext(ctx, `
		SELECT u.id, COALESCE(u.username, ''), u.display_name,
		       p.avatar_url, p.banner_url, p.bio,
		       COALESCE(p.visibility, 'public'), u.created_at,
		       (SELECT COUNT(*) FROM follows f WHERE f.followed_id = u.id),
		       (SELECT COUNT(*) FROM follows f2 WHERE f2.follower_id = u.id),
		       (SELECT COUNT(*) FROM posts po WHERE po.author_id = u.id AND po.deleted_at IS NULL),
		       EXISTS(SELECT 1 FROM follows f3 WHERE f3.follower_id = ? AND f3.followed_id = u.id)
		FROM users u
		LEFT JOIN user_profiles p ON p.user_id = u.id
		WHERE u.id <> ?
		  AND u.is_active = 1
		  AND u.discoverable = 1
		  AND COALESCE(p.visibility, 'public') <> 'private'
		  AND NOT EXISTS (
		      SELECT 1 FROM blocks b
		      WHERE (b.blocker_id = u.id AND b.blocked_id = ?)
		         OR (b.blocker_id = ? AND b.blocked_id = u.id))
		  AND NOT EXISTS (
		      SELECT 1 FROM discovery_suppressions s
		      WHERE s.viewer_id = ? AND s.candidate_id = u.id)
		  AND NOT EXISTS (
		      SELECT 1 FROM connections c
		      WHERE c.status = 'accepted'
		        AND ((c.requester_id = ? AND c.addressee_id = u.id)
		          OR (c.addressee_id = ? AND c.requester_id = u.id)))
		ORDER BY u.id
		LIMIT ?`,
		viewerID, viewerID, viewerID, viewerID, viewerID, viewerID, viewerID, candidateCap)
	if err != nil {
		return nil, fmt.Errorf("load discovery candidates: %w", err)
	}
	defer rows.Close()

	out := make([]candidateRow, 0)
	for rows.Next() {
		var candidate candidateRow
		var avatar, banner, bio sql.NullString
		var visibility, createdAt string
		var followers, following, posts int64
		var isFollowing bool
		if err := rows.Scan(&candidate.user.ID, &candidate.user.Username, &candidate.user.DisplayName,
			&avatar, &banner, &bio, &visibility, &createdAt,
			&followers, &following, &posts, &isFollowing); err != nil {
			return nil, fmt.Errorf("scan discovery candidate: %w", err)
		}
		candidate.user.AvatarURL = nullString(avatar)
		candidate.user.BannerURL = nullString(banner)
		candidate.user.Bio = nullString(bio)
		candidate.user.Visibility = visibility
		candidate.user.Discoverable = true
		candidate.user.CreatedAt = createdAt
		candidate.user.Interests = []Interest{}
		candidate.user.AnimeInterests = []AnimeInterest{}
		candidate.user.FollowerCount = followers
		candidate.user.FollowingCount = following
		candidate.user.PostCount = posts
		candidate.user.IsFollowing = isFollowing
		out = append(out, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate discovery candidates: %w", err)
	}
	return out, nil
}

type explanationGroup struct {
	count    int
	priority int
	verb     string
	plural   string
	singular string
}

func (g explanationGroup) nounPhrase() string {
	noun := g.plural
	if g.count == 1 {
		noun = g.singular
	}
	return fmt.Sprintf("%d of the same %s", g.count, noun)
}

func (g explanationGroup) clause() string {
	return g.verb + " " + g.nounPhrase()
}

// buildExplanation uses the two largest overlap groups and a plain sentence.
// Zero groups are omitted; no scores or percentages are ever rendered.
func buildExplanation(groups []explanationGroup) string {
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].count != groups[j].count {
			return groups[i].count > groups[j].count
		}
		return groups[i].priority < groups[j].priority
	})
	top := groups
	if len(top) > 2 {
		top = top[:2]
	}
	if len(top) == 0 {
		return ""
	}
	sentence := "You both " + top[0].clause()
	if len(top) == 2 {
		sentence += " and " + top[1].nounPhrase()
	}
	return sentence
}

// decodeScoreCursor reads the opaque "count|id" pagination cursor.
func decodeScoreCursor(encoded string) (count int, id int64, present bool, valid bool) {
	raw, err := shared.DecodeCursor(encoded)
	if err != nil {
		return 0, 0, false, false
	}
	if raw == "" {
		return 0, 0, false, true
	}
	parts := strings.Split(raw, "|")
	if len(parts) != 2 {
		return 0, 0, true, false
	}
	count, err = strconv.Atoi(parts[0])
	if err != nil || count < 0 {
		return 0, 0, true, false
	}
	id, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil || id <= 0 {
		return 0, 0, true, false
	}
	return count, id, true, true
}

func (m *module) handleFriends(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := requireViewer(w, r, m)
	if !ok {
		return
	}
	pg := shared.ParsePagination(r)
	cursorCount, cursorID, hasCursor, valid := decodeScoreCursor(pg.Cursor)
	if !valid {
		writeValidation(w, "Invalid pagination cursor.", fieldErrors{
			"cursor": "must be a valid discovery cursor",
		})
		return
	}

	sets, err := m.loadViewerSets(r.Context(), viewerID)
	if err != nil {
		internalError(w, m.d, "load viewer interests", err)
		return
	}
	candidates, err := m.loadCandidates(r.Context(), viewerID)
	if err != nil {
		internalError(w, m.d, "load candidates", err)
		return
	}
	ids := make([]int64, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.user.ID)
	}
	signals, err := m.loadCandidateSignals(r.Context(), ids)
	if err != nil {
		internalError(w, m.d, "load candidate interests", err)
		return
	}
	statuses, err := m.loadCandidateConnections(r.Context(), viewerID, ids)
	if err != nil {
		internalError(w, m.d, "load candidate connections", err)
		return
	}

	scored := make([]DiscoveryCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		signal := signals[candidate.user.ID]
		if signal == nil {
			signal = newCandidateSignals()
		}
		anime := intersectNames(sets.anime, signal.anime)
		genres := intersectNames(sets.genres, signal.genres)
		interests := intersectNames(sets.interests, signal.interests)
		communities := intersectNames(sets.communities, signal.communities)
		creatorOverlap := intersectCategories(sets.creators, signal.creators)

		groups := make([]explanationGroup, 0, 5)
		if len(anime) > 0 {
			groups = append(groups, explanationGroup{len(anime), 0, "follow", "anime", "anime"})
		}
		if len(genres) > 0 {
			groups = append(groups, explanationGroup{len(genres), 1, "like", "genres", "genre"})
		}
		if len(interests) > 0 {
			groups = append(groups, explanationGroup{len(interests), 2, "share", "interests", "interest"})
		}
		if len(communities) > 0 {
			groups = append(groups, explanationGroup{len(communities), 3, "follow", "communities", "community"})
		}
		if creatorOverlap > 0 {
			groups = append(groups, explanationGroup{creatorOverlap, 4, "create in", "categories", "category"})
		}

		candidate.user.Interests = signal.interestDTO
		candidate.user.AnimeInterests = signal.animeDTO
		status := statuses[candidate.user.ID]
		if status == "" {
			status = "none"
		}
		scored = append(scored, DiscoveryCandidate{
			User:             candidate.user,
			Shared:           SharedInterests{Anime: capNames(anime), Genres: capNames(genres), Interests: capNames(interests), Communities: capNames(communities)},
			SharedCount:      len(anime) + len(genres) + len(interests) + len(communities) + creatorOverlap,
			Explanation:      buildExplanation(groups),
			ConnectionStatus: status,
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].SharedCount != scored[j].SharedCount {
			return scored[i].SharedCount > scored[j].SharedCount
		}
		return scored[i].User.ID < scored[j].User.ID
	})

	start := 0
	if hasCursor {
		for start < len(scored) {
			item := scored[start]
			if item.SharedCount < cursorCount ||
				(item.SharedCount == cursorCount && item.User.ID > cursorID) {
				break
			}
			start++
		}
	}
	window := scored[start:]
	nextCursor := ""
	if len(window) > pg.Limit {
		last := window[pg.Limit-1]
		nextCursor = shared.EncodeCursor(fmt.Sprintf("%d|%d", last.SharedCount, last.User.ID))
		window = window[:pg.Limit]
	}
	shared.WriteJSON(w, http.StatusOK, page[DiscoveryCandidate]{Items: window, NextCursor: nextCursor})
}
