package users

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

func decodeJSONRaw(w http.ResponseWriter, r *http.Request) (json.RawMessage, bool) {
	var raw json.RawMessage
	if !shared.DecodeJSON(w, r, &raw) {
		return nil, false
	}
	return raw, true
}

func decodeInterestIDs(w http.ResponseWriter, r *http.Request) ([]int64, bool) {
	raw, ok := decodeJSONRaw(w, r)
	if !ok {
		return nil, false
	}
	var ids []int64
	if err := json.Unmarshal(raw, &ids); err == nil {
		return ids, true
	}
	var wrapped struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil {
		return wrapped.IDs, true
	}
	shared.WriteError(w, http.StatusBadRequest, shared.CodeInvalidJSON,
		"Body must be an array of interest ids.")
	return nil, false
}

func decodeAnimeInterests(w http.ResponseWriter, r *http.Request) ([]animeInterestInput, bool) {
	raw, ok := decodeJSONRaw(w, r)
	if !ok {
		return nil, false
	}
	var items []animeInterestInput
	if err := json.Unmarshal(raw, &items); err == nil {
		return items, true
	}
	var wrapped struct {
		Items []animeInterestInput `json:"items"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil {
		return wrapped.Items, true
	}
	shared.WriteError(w, http.StatusBadRequest, shared.CodeInvalidJSON,
		"Body must be an array of {animeId, interestType} objects.")
	return nil, false
}

func (s *service) listInterests(w http.ResponseWriter, r *http.Request) {
	rows, err := s.d.DB.QueryContext(r.Context(),
		`SELECT id, name, category FROM interests ORDER BY category, name, id`)
	if err != nil {
		internalError(w, s.d, "list interests", err)
		return
	}
	defer func() { _ = rows.Close() }()
	interests := []Interest{}
	for rows.Next() {
		var interest Interest
		if err := rows.Scan(&interest.ID, &interest.Name, &interest.Category); err != nil {
			internalError(w, s.d, "list interests: scan", err)
			return
		}
		interests = append(interests, interest)
	}
	if err := rows.Err(); err != nil {
		internalError(w, s.d, "list interests: iterate", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, interests)
}

func (s *service) replaceInterests(w http.ResponseWriter, r *http.Request) {
	userID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	ids, ok := decodeInterestIDs(w, r)
	if !ok {
		return
	}
	details := make(validationErrors)
	unique := make([]int64, 0, len(ids))
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id <= 0 {
			details.add("ids", "Every interest id must be a positive integer.")
			continue
		}
		if seen[id] {
			continue // duplicate ids are ignored, not an error
		}
		seen[id] = true
		unique = append(unique, id)
	}
	if len(details) > 0 {
		writeValidation(w, details)
		return
	}

	tx, err := s.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		internalError(w, s.d, "replace interests: begin transaction", err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	for _, id := range unique {
		var exists int
		err := tx.QueryRowContext(r.Context(),
			`SELECT 1 FROM interests WHERE id = ? LIMIT 1`, id).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			writeValidation(w, validationErrors{"ids": "Unknown interest id."})
			return
		}
		if err != nil {
			internalError(w, s.d, "replace interests: check id", err)
			return
		}
	}
	if _, err := tx.ExecContext(r.Context(),
		`DELETE FROM user_interests WHERE user_id = ?`, userID); err != nil {
		internalError(w, s.d, "replace interests: delete old", err)
		return
	}
	for index, id := range unique {
		if _, err := tx.ExecContext(r.Context(), `
			INSERT INTO user_interests (user_id, interest_id, sort_order)
			VALUES (?, ?, ?)`, userID, id, index); err != nil {
			internalError(w, s.d, "replace interests: insert", err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		internalError(w, s.d, "replace interests: commit", err)
		return
	}
	profile, err := s.loadProfile(r.Context(), userID, userID)
	if err != nil {
		internalError(w, s.d, "replace interests: reload profile", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, profile)
}

func (s *service) replaceAnimeInterests(w http.ResponseWriter, r *http.Request) {
	userID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	items, ok := decodeAnimeInterests(w, r)
	if !ok {
		return
	}
	details := make(validationErrors)
	unique := make([]animeInterestInput, 0, len(items))
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		itemType := strings.ToLower(strings.TrimSpace(item.InterestType))
		if item.AnimeID <= 0 || itemType == "" || len(itemType) > 40 {
			details.add("items", "Each item needs a valid animeId and interestType.")
			continue
		}
		key := fmt.Sprintf("%s:%d", itemType, item.AnimeID)
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, animeInterestInput{AnimeID: item.AnimeID, InterestType: itemType})
	}
	if len(details) > 0 {
		writeValidation(w, details)
		return
	}

	tx, err := s.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		internalError(w, s.d, "replace anime interests: begin transaction", err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	for _, item := range unique {
		var exists int
		err := tx.QueryRowContext(r.Context(),
			`SELECT 1 FROM anime_titles WHERE id = ? LIMIT 1`, item.AnimeID).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			writeValidation(w, validationErrors{"items": "Unknown anime id."})
			return
		}
		if err != nil {
			internalError(w, s.d, "replace anime interests: check id", err)
			return
		}
	}
	if _, err := tx.ExecContext(r.Context(),
		`DELETE FROM anime_interests WHERE user_id = ?`, userID); err != nil {
		internalError(w, s.d, "replace anime interests: delete old", err)
		return
	}
	for index, item := range unique {
		if _, err := tx.ExecContext(r.Context(), `
			INSERT INTO anime_interests (user_id, anime_title_id, interest_type, sort_order)
			VALUES (?, ?, ?, ?)`, userID, item.AnimeID, item.InterestType, index); err != nil {
			internalError(w, s.d, "replace anime interests: insert", err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		internalError(w, s.d, "replace anime interests: commit", err)
		return
	}
	profile, err := s.loadProfile(r.Context(), userID, userID)
	if err != nil {
		internalError(w, s.d, "replace anime interests: reload profile", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, profile)
}
