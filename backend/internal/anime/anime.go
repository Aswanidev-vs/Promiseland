// Package anime exposes the small seeded anime catalogue and genre list.
package anime

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

// Anime and Genre match the R1 wire shapes from docs/conventions.md. Anime
// fields are plain strings: seeded rows without a verified external catalogue
// id expose an empty string rather than a fabricated one.
type Anime struct {
	ID             int64  `json:"id"`
	ExternalID     string `json:"external_id"`
	Title          string `json:"title"`
	AlternateTitle string `json:"alternate_title"`
	Format         string `json:"format"`
}

type Genre struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Page is the R1 envelope: next_cursor is always present and is "" when there
// are no more results.
type Page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor"`
}

const (
	codeInvalidID     = "invalid_id"
	codeInvalidCursor = "invalid_cursor"
)

// likeEscaper prevents user-supplied % and _ from acting as LIKE wildcards.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

type service struct {
	d *shared.Deps
}

// Register mounts the public anime and genre routes under /api/v1.
func Register(r chi.Router, d *shared.Deps) {
	s := &service{d: d}
	r.Get("/anime", s.search)
	r.Get("/anime/{id}", s.getByID)
	r.Get("/genres", s.genres)
}

// RegisterProtected exists for symmetry with the other R1 modules. Anime
// endpoints are read-only and need no authenticated routes.
func RegisterProtected(r chi.Router, d *shared.Deps) {}

func internalError(w http.ResponseWriter, d *shared.Deps, action string, err error) {
	if d != nil && d.Log != nil {
		d.Log.Error("anime: "+action, "error", err)
	}
	shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError,
		"An unexpected error occurred.")
}

func scanAnime(scanner interface{ Scan(...any) error }) (Anime, error) {
	var anime Anime
	err := scanner.Scan(
		&anime.ID, &anime.ExternalID, &anime.Title,
		&anime.AlternateTitle, &anime.Format,
	)
	return anime, err
}

const animeColumns = `
	SELECT id, COALESCE(external_id, ''), title,
	       COALESCE(alternate_title, ''), COALESCE(format, '')
	FROM anime_titles`

func (s *service) search(w http.ResponseWriter, r *http.Request) {
	pagination := shared.ParsePagination(r)
	var cursor int64
	if pagination.Cursor != "" {
		var err error
		cursor, err = shared.DecodeIDCursor(pagination.Cursor)
		if err != nil {
			shared.WriteError(w, http.StatusBadRequest, codeInvalidCursor, "Invalid cursor.")
			return
		}
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	pattern := "%" + likeEscaper.Replace(query) + "%"

	rows, err := s.d.DB.QueryContext(r.Context(), animeColumns+`
		WHERE (? = '' OR lower(title) LIKE ? ESCAPE '\'
		              OR lower(COALESCE(alternate_title, '')) LIKE ? ESCAPE '\')
		  AND id > ?
		ORDER BY id
		LIMIT ?`, query, pattern, pattern, cursor, pagination.Limit+1)
	if err != nil {
		internalError(w, s.d, "search", err)
		return
	}
	defer func() { _ = rows.Close() }()
	items := []Anime{}
	for rows.Next() {
		anime, err := scanAnime(rows)
		if err != nil {
			internalError(w, s.d, "search: scan", err)
			return
		}
		items = append(items, anime)
	}
	if err := rows.Err(); err != nil {
		internalError(w, s.d, "search: iterate", err)
		return
	}

	result := shared.NewPage(items)
	page := Page[Anime]{Items: result.Items}
	if len(items) > pagination.Limit {
		page.Items = items[:pagination.Limit]
		page.NextCursor = shared.EncodeIDCursor(page.Items[len(page.Items)-1].ID)
	}
	shared.WriteJSON(w, http.StatusOK, page)
}

func (s *service) getByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		shared.WriteError(w, http.StatusBadRequest, codeInvalidID, "Invalid anime id.")
		return
	}
	anime, err := scanAnime(s.d.DB.QueryRowContext(r.Context(), animeColumns+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Anime title not found.")
		return
	}
	if err != nil {
		internalError(w, s.d, "get by id", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, anime)
}

func (s *service) genres(w http.ResponseWriter, r *http.Request) {
	s.listGenres(r.Context(), w)
}

func (s *service) listGenres(ctx context.Context, w http.ResponseWriter) {
	rows, err := s.d.DB.QueryContext(ctx, `SELECT id, name FROM genres ORDER BY name, id`)
	if err != nil {
		internalError(w, s.d, "list genres", err)
		return
	}
	defer func() { _ = rows.Close() }()
	genres := []Genre{}
	for rows.Next() {
		var genre Genre
		if err := rows.Scan(&genre.ID, &genre.Name); err != nil {
			internalError(w, s.d, "list genres: scan", err)
			return
		}
		genres = append(genres, genre)
	}
	if err := rows.Err(); err != nil {
		internalError(w, s.d, "list genres: iterate", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, genres)
}
