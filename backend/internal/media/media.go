// Package media handles validated image uploads and controlled static serving.
// Image bytes live on disk under MEDIA_PATH; SQLite stores only metadata.
package media

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
	_ "golang.org/x/image/webp" // registers the webp decoder for image.DecodeConfig
)

const (
	maxUploadBytes = 5 << 20 // 5 MiB of image data
	maxMultipart   = maxUploadBytes + (1 << 20)
	maxDimension   = 6000
	// maxUploadsPerHour bounds how fast one account can fill the volume that
	// also holds the database: an unbounded loop of valid uploads would
	// otherwise starve SQLite writes for everyone.
	maxUploadsPerHour = 30
)

var allowedFormats = map[string]struct {
	mime   string
	format string
}{
	".jpg":  {"image/jpeg", "jpeg"},
	".jpeg": {"image/jpeg", "jpeg"},
	".png":  {"image/png", "png"},
	".gif":  {"image/gif", "gif"},
	".webp": {"image/webp", "webp"},
}

// Media is the exact R1 media shape.
type Media struct {
	ID        int64  `json:"id"`
	URL       string `json:"url"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

// Register mounts public /api/v1 media routes. Media has no public API read
// endpoint in R1; serving is handled by RegisterMedia on the root router.
func Register(r chi.Router, d *shared.Deps) {}

// RegisterProtected mounts the authenticated upload endpoint.
func RegisterProtected(r chi.Router, d *shared.Deps) {
	h := &handler{d: d}
	r.Post("/media", h.upload)
}

// RegisterMedia mounts controlled GET /media/{key} static serving on the root
// router (outside /api/v1).
func RegisterMedia(r chi.Router, d *shared.Deps) {
	h := &handler{d: d}
	r.Get("/media/{key}", h.serve)
}

type handler struct{ d *shared.Deps }

func mediaRoot() string {
	root := strings.TrimSpace(os.Getenv("MEDIA_PATH"))
	if root == "" {
		return filepath.Join(".", "data", "media")
	}
	return root
}

func writeUploadError(w http.ResponseWriter, status int, field, message string) {
	shared.WriteErrorDetails(w, status, "invalid_file", "The file could not be uploaded.", map[string]string{field: message})
}

func (h *handler) upload(w http.ResponseWriter, r *http.Request) {
	userID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	var recent int
	if err := h.d.DB.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM media WHERE owner_id = ? AND created_at > datetime('now','-1 hour')`,
		userID).Scan(&recent); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not check the upload allowance.")
		return
	}
	if recent >= maxUploadsPerHour {
		w.Header().Set("Retry-After", "600")
		shared.WriteError(w, http.StatusTooManyRequests, "rate_limited",
			"Too many uploads in a short time. Wait a few minutes and try again.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxMultipart)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writeUploadError(w, http.StatusBadRequest, "file", "Files must be 5 MB or smaller.")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeUploadError(w, http.StatusBadRequest, "file", "Attach an image file named 'file'.")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	expected, ok := allowedFormats[ext]
	if !ok {
		writeUploadError(w, http.StatusBadRequest, "file", "Use a JPEG, PNG, GIF or WebP image.")
		return
	}
	declared := strings.ToLower(strings.TrimSpace(strings.Split(header.Header.Get("Content-Type"), ";")[0]))
	if declared != expected.mime {
		writeUploadError(w, http.StatusBadRequest, "file", "The declared file type does not match its extension.")
		return
	}

	data, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil {
		writeUploadError(w, http.StatusBadRequest, "file", "The file could not be read.")
		return
	}
	if len(data) > maxUploadBytes {
		writeUploadError(w, http.StatusBadRequest, "file", "Files must be 5 MB or smaller.")
		return
	}
	if len(data) == 0 {
		writeUploadError(w, http.StatusBadRequest, "file", "The file is empty.")
		return
	}
	// DetectContentType verifies actual bytes, not just the client's claim.
	detected := http.DetectContentType(data)
	if detected != expected.mime {
		writeUploadError(w, http.StatusBadRequest, "file", "File contents do not match the declared image type.")
		return
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != expected.format {
		writeUploadError(w, http.StatusBadRequest, "file", "The image could not be decoded.")
		return
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > maxDimension || config.Height > maxDimension {
		writeUploadError(w, http.StatusBadRequest, "file", "Image dimensions must be positive and at most 6000 pixels.")
		return
	}

	var raw [24]byte
	if _, err := rand.Read(raw[:]); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not store the upload.")
		return
	}
	key := hex.EncodeToString(raw[:]) + ext
	root := mediaRoot()
	if err := os.MkdirAll(root, 0o755); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not store the upload.")
		return
	}
	path := filepath.Join(root, key)
	tmp := path + ".uploading"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not store the upload.")
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not store the upload.")
		return
	}

	res, err := h.d.DB.ExecContext(r.Context(), `
		INSERT INTO media(owner_id, storage_key, mime_type, size_bytes, width, height, created_at)
		VALUES(?,?,?,?,?,?,datetime('now'))`,
		userID, key, expected.mime, len(data), config.Width, config.Height)
	if err != nil {
		// No metadata row means no broken feed entry and no orphan bytes.
		_ = os.Remove(path)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not store the upload.")
		return
	}
	id, _ := res.LastInsertId()
	shared.WriteJSON(w, http.StatusCreated, Media{
		ID: id, URL: "/media/" + key, MimeType: expected.mime,
		SizeBytes: int64(len(data)), Width: config.Width, Height: config.Height,
	})
}

func (h *handler) serve(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" || strings.Contains(key, "..") || strings.ContainsAny(key, `/\`) ||
		filepath.Base(key) != key || strings.HasPrefix(key, ".") {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Media not found.")
		return
	}
	var mimeType string
	err := h.d.DB.QueryRowContext(r.Context(), `SELECT mime_type FROM media WHERE storage_key = ?`, key).Scan(&mimeType)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Media not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the media.")
		return
	}
	path := filepath.Join(mediaRoot(), key)
	if _, err := os.Stat(path); err != nil {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Media not found.")
		return
	}
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeFile(w, r, path)
}
