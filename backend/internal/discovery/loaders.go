package discovery

import (
	"context"
	"fmt"
	"sort"
)

// interestSets is the viewer's interest profile. IDs are the keys so overlap
// counting is a map lookup; the values are readable names for explanations.
type interestSets struct {
	anime       map[int64]string
	genres      map[int64]string
	interests   map[int64]string
	communities map[int64]string
	creators    map[string]struct{}
}

func newInterestSets() interestSets {
	return interestSets{
		anime:       map[int64]string{},
		genres:      map[int64]string{},
		interests:   map[int64]string{},
		communities: map[int64]string{},
		creators:    map[string]struct{}{},
	}
}

// loadViewerSets builds the viewer's overlap sets with one query per signal.
func (m *module) loadViewerSets(ctx context.Context, viewerID int64) (interestSets, error) {
	sets := newInterestSets()

	animeRows, err := m.d.DB.QueryContext(ctx, `
		SELECT ai.anime_title_id, a.title
		FROM anime_interests ai
		JOIN anime_titles a ON a.id = ai.anime_title_id
		WHERE ai.user_id = ?`, viewerID)
	if err != nil {
		return sets, fmt.Errorf("load viewer anime: %w", err)
	}
	for animeRows.Next() {
		var id int64
		var title string
		if err := animeRows.Scan(&id, &title); err != nil {
			_ = animeRows.Close()
			return sets, fmt.Errorf("scan viewer anime: %w", err)
		}
		sets.anime[id] = title
	}
	if err := animeRows.Err(); err != nil {
		_ = animeRows.Close()
		return sets, fmt.Errorf("iterate viewer anime: %w", err)
	}
	_ = animeRows.Close()

	genreRows, err := m.d.DB.QueryContext(ctx, `
		SELECT DISTINCT ag.genre_id, g.name
		FROM anime_genres ag
		JOIN genres g ON g.id = ag.genre_id
		JOIN anime_interests ai ON ai.anime_title_id = ag.anime_id
		WHERE ai.user_id = ?`, viewerID)
	if err != nil {
		return sets, fmt.Errorf("load viewer genres: %w", err)
	}
	for genreRows.Next() {
		var id int64
		var name string
		if err := genreRows.Scan(&id, &name); err != nil {
			_ = genreRows.Close()
			return sets, fmt.Errorf("scan viewer genres: %w", err)
		}
		sets.genres[id] = name
	}
	if err := genreRows.Err(); err != nil {
		_ = genreRows.Close()
		return sets, fmt.Errorf("iterate viewer genres: %w", err)
	}
	_ = genreRows.Close()

	interestRows, err := m.d.DB.QueryContext(ctx, `
		SELECT i.id, i.name
		FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ?`, viewerID)
	if err != nil {
		return sets, fmt.Errorf("load viewer interests: %w", err)
	}
	for interestRows.Next() {
		var id int64
		var name string
		if err := interestRows.Scan(&id, &name); err != nil {
			_ = interestRows.Close()
			return sets, fmt.Errorf("scan viewer interests: %w", err)
		}
		sets.interests[id] = name
	}
	if err := interestRows.Err(); err != nil {
		_ = interestRows.Close()
		return sets, fmt.Errorf("iterate viewer interests: %w", err)
	}
	_ = interestRows.Close()

	communityRows, err := m.d.DB.QueryContext(ctx, `
		SELECT cm.community_id, c.name
		FROM community_memberships cm
		JOIN communities c ON c.id = cm.community_id
		WHERE cm.user_id = ? AND cm.status = 'active'`, viewerID)
	if err != nil {
		return sets, fmt.Errorf("load viewer communities: %w", err)
	}
	for communityRows.Next() {
		var id int64
		var name string
		if err := communityRows.Scan(&id, &name); err != nil {
			_ = communityRows.Close()
			return sets, fmt.Errorf("scan viewer communities: %w", err)
		}
		sets.communities[id] = name
	}
	if err := communityRows.Err(); err != nil {
		_ = communityRows.Close()
		return sets, fmt.Errorf("iterate viewer communities: %w", err)
	}
	_ = communityRows.Close()

	creatorRows, err := m.d.DB.QueryContext(ctx, `
		SELECT DISTINCT creator_category
		FROM posts
		WHERE author_id = ? AND creator_category <> '' AND deleted_at IS NULL`, viewerID)
	if err != nil {
		return sets, fmt.Errorf("load viewer creator categories: %w", err)
	}
	for creatorRows.Next() {
		var category string
		if err := creatorRows.Scan(&category); err != nil {
			_ = creatorRows.Close()
			return sets, fmt.Errorf("scan viewer creator categories: %w", err)
		}
		sets.creators[category] = struct{}{}
	}
	if err := creatorRows.Err(); err != nil {
		_ = creatorRows.Close()
		return sets, fmt.Errorf("iterate viewer creator categories: %w", err)
	}
	_ = creatorRows.Close()

	return sets, nil
}

// candidateSignals holds all overlap-relevant data and the full R1 arrays for
// one candidate user.
type candidateSignals struct {
	anime       map[int64]string
	genres      map[int64]string
	interests   map[int64]string
	communities map[int64]string
	creators    map[string]struct{}
	interestDTO []Interest
	animeDTO    []AnimeInterest
}

func newCandidateSignals() *candidateSignals {
	return &candidateSignals{
		anime:       map[int64]string{},
		genres:      map[int64]string{},
		interests:   map[int64]string{},
		communities: map[int64]string{},
		creators:    map[string]struct{}{},
		interestDTO: []Interest{},
		animeDTO:    []AnimeInterest{},
	}
}

// loadCandidateSignals loads overlap data for the capped candidate set in
// bounded chunks, never one query per candidate.
func (m *module) loadCandidateSignals(ctx context.Context, ids []int64) (map[int64]*candidateSignals, error) {
	result := make(map[int64]*candidateSignals, len(ids))
	for _, id := range ids {
		result[id] = newCandidateSignals()
	}
	if len(ids) == 0 {
		return result, nil
	}

	ensure := func(userID int64) *candidateSignals {
		signals, ok := result[userID]
		if !ok {
			signals = newCandidateSignals()
			result[userID] = signals
		}
		return signals
	}

	for _, chunk := range chunkIDs(ids) {
		in := placeholders(len(chunk))
		args := anyIDs(chunk)

		animeRows, err := m.d.DB.QueryContext(ctx, fmt.Sprintf(`
			SELECT ai.user_id, ai.anime_title_id, a.title,
			       COALESCE(a.external_id, ''), COALESCE(a.alternate_title, ''),
			       COALESCE(a.format, ''), ai.interest_type, ai.sort_order
			FROM anime_interests ai
			JOIN anime_titles a ON a.id = ai.anime_title_id
			WHERE ai.user_id IN (%s)
			ORDER BY ai.user_id, ai.sort_order, ai.anime_title_id, ai.interest_type`, in), args...)
		if err != nil {
			return nil, fmt.Errorf("load candidate anime: %w", err)
		}
		for animeRows.Next() {
			var userID, animeID int64
			var title, externalID, alternateTitle, format, interestType string
			var sortOrder int
			if err := animeRows.Scan(&userID, &animeID, &title, &externalID, &alternateTitle,
				&format, &interestType, &sortOrder); err != nil {
				_ = animeRows.Close()
				return nil, fmt.Errorf("scan candidate anime: %w", err)
			}
			signals := ensure(userID)
			signals.anime[animeID] = title
			signals.animeDTO = append(signals.animeDTO, AnimeInterest{
				Anime: Anime{
					ID:             animeID,
					ExternalID:     externalID,
					Title:          title,
					AlternateTitle: alternateTitle,
					Format:         format,
				},
				InterestType: interestType,
				SortOrder:    sortOrder,
			})
		}
		if err := animeRows.Err(); err != nil {
			_ = animeRows.Close()
			return nil, fmt.Errorf("iterate candidate anime: %w", err)
		}
		_ = animeRows.Close()

		genreRows, err := m.d.DB.QueryContext(ctx, fmt.Sprintf(`
			SELECT DISTINCT ai.user_id, ag.genre_id, g.name
			FROM anime_interests ai
			JOIN anime_genres ag ON ag.anime_id = ai.anime_title_id
			JOIN genres g ON g.id = ag.genre_id
			WHERE ai.user_id IN (%s)`, in), args...)
		if err != nil {
			return nil, fmt.Errorf("load candidate genres: %w", err)
		}
		for genreRows.Next() {
			var userID, genreID int64
			var name string
			if err := genreRows.Scan(&userID, &genreID, &name); err != nil {
				_ = genreRows.Close()
				return nil, fmt.Errorf("scan candidate genres: %w", err)
			}
			ensure(userID).genres[genreID] = name
		}
		if err := genreRows.Err(); err != nil {
			_ = genreRows.Close()
			return nil, fmt.Errorf("iterate candidate genres: %w", err)
		}
		_ = genreRows.Close()

		interestRows, err := m.d.DB.QueryContext(ctx, fmt.Sprintf(`
			SELECT ui.user_id, i.id, i.name, i.category
			FROM user_interests ui
			JOIN interests i ON i.id = ui.interest_id
			WHERE ui.user_id IN (%s)
			ORDER BY ui.user_id, ui.sort_order, i.id`, in), args...)
		if err != nil {
			return nil, fmt.Errorf("load candidate interests: %w", err)
		}
		for interestRows.Next() {
			var userID, interestID int64
			var name, category string
			if err := interestRows.Scan(&userID, &interestID, &name, &category); err != nil {
				_ = interestRows.Close()
				return nil, fmt.Errorf("scan candidate interests: %w", err)
			}
			signals := ensure(userID)
			signals.interests[interestID] = name
			signals.interestDTO = append(signals.interestDTO, Interest{
				ID: interestID, Name: name, Category: category,
			})
		}
		if err := interestRows.Err(); err != nil {
			_ = interestRows.Close()
			return nil, fmt.Errorf("iterate candidate interests: %w", err)
		}
		_ = interestRows.Close()

		communityRows, err := m.d.DB.QueryContext(ctx, fmt.Sprintf(`
			SELECT cm.user_id, cm.community_id, c.name
			FROM community_memberships cm
			JOIN communities c ON c.id = cm.community_id
			WHERE cm.user_id IN (%s) AND cm.status = 'active'
			ORDER BY cm.user_id, cm.community_id`, in), args...)
		if err != nil {
			return nil, fmt.Errorf("load candidate communities: %w", err)
		}
		for communityRows.Next() {
			var userID, communityID int64
			var name string
			if err := communityRows.Scan(&userID, &communityID, &name); err != nil {
				_ = communityRows.Close()
				return nil, fmt.Errorf("scan candidate communities: %w", err)
			}
			ensure(userID).communities[communityID] = name
		}
		if err := communityRows.Err(); err != nil {
			_ = communityRows.Close()
			return nil, fmt.Errorf("iterate candidate communities: %w", err)
		}
		_ = communityRows.Close()

		creatorRows, err := m.d.DB.QueryContext(ctx, fmt.Sprintf(`
			SELECT DISTINCT author_id, creator_category
			FROM posts
			WHERE author_id IN (%s) AND creator_category <> '' AND deleted_at IS NULL`, in), args...)
		if err != nil {
			return nil, fmt.Errorf("load candidate creator categories: %w", err)
		}
		for creatorRows.Next() {
			var userID int64
			var category string
			if err := creatorRows.Scan(&userID, &category); err != nil {
				_ = creatorRows.Close()
				return nil, fmt.Errorf("scan candidate creator categories: %w", err)
			}
			ensure(userID).creators[category] = struct{}{}
		}
		if err := creatorRows.Err(); err != nil {
			_ = creatorRows.Close()
			return nil, fmt.Errorf("iterate candidate creator categories: %w", err)
		}
		_ = creatorRows.Close()
	}
	return result, nil
}

// loadCandidateConnections returns the viewer-facing status label for each
// candidate: pending_in, pending_out or accepted.
func (m *module) loadCandidateConnections(ctx context.Context, viewerID int64, ids []int64) (map[int64]string, error) {
	statuses := map[int64]string{}
	if len(ids) == 0 {
		return statuses, nil
	}
	for _, chunk := range chunkIDs(ids) {
		in := placeholders(len(chunk))
		args := make([]any, 0, 2*len(chunk)+2)
		args = append(args, viewerID)
		args = append(args, anyIDs(chunk)...)
		args = append(args, viewerID)
		args = append(args, anyIDs(chunk)...)
		rows, err := m.d.DB.QueryContext(ctx, fmt.Sprintf(`
			SELECT requester_id, addressee_id, status
			FROM connections
			WHERE (requester_id = ? AND addressee_id IN (%s))
			   OR (addressee_id = ? AND requester_id IN (%s))`, in, in), args...)
		if err != nil {
			return nil, fmt.Errorf("load candidate connections: %w", err)
		}
		for rows.Next() {
			var requesterID, addresseeID int64
			var status string
			if err := rows.Scan(&requesterID, &addresseeID, &status); err != nil {
				_ = rows.Close()
				return nil, fmt.Errorf("scan candidate connection: %w", err)
			}
			other := addresseeID
			label := "pending_out"
			if requesterID != viewerID {
				other = requesterID
				label = "pending_in"
			}
			if status == "accepted" {
				label = "accepted"
			}
			// An incoming request wins when both directions exist in old data.
			if _, exists := statuses[other]; !exists || label == "pending_in" {
				statuses[other] = label
			}
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("iterate candidate connections: %w", err)
		}
		_ = rows.Close()
	}
	return statuses, nil
}

// intersectNames returns the sorted, readable names shared by two sets.
func intersectNames(viewer, candidate map[int64]string) []string {
	names := make([]string, 0, len(candidate))
	for id, name := range candidate {
		if _, ok := viewer[id]; ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func intersectCategories(viewer, candidate map[string]struct{}) int {
	count := 0
	for category := range candidate {
		if _, ok := viewer[category]; ok {
			count++
		}
	}
	return count
}

func capNames(names []string) []string {
	if len(names) <= sharedNameCap {
		return names
	}
	return names[:sharedNameCap]
}
