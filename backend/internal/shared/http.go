package shared

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Canonical API error codes shared by all modules.
const (
	CodeInvalidJSON      = "invalid_json"
	CodeNotFound         = "not_found"
	CodeMethodNotAllowed = "method_not_allowed"
	CodeAuthRequired     = "auth_required"
	CodeInternalError    = "internal_error"
	CodeDBUnavailable    = "database_unavailable"
)

const (
	// DefaultLimit is the default page size.
	DefaultLimit = 20
	// MaxLimit caps client-requested page sizes.
	MaxLimit = 100
	// MaxCursorLen bounds both raw and encoded cursor values.
	MaxCursorLen = 512
	// maxBodyBytes caps JSON request bodies at 1 MB.
	maxBodyBytes = 1 << 20
)

type errorResponse struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// WriteJSON writes v as JSON with the canonical content type.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(v)
}

// WriteError writes the canonical error envelope:
// {"error":{"code":"...","message":"..."}}.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, errorResponse{Error: apiError{Code: code, Message: message}})
}

// WriteErrorDetails is WriteError with optional structured details (for
// example, per-field validation errors).
func WriteErrorDetails(w http.ResponseWriter, status int, code, message string, details any) {
	WriteJSON(w, status, errorResponse{Error: apiError{Code: code, Message: message, Details: details}})
}

// DecodeJSON decodes a JSON request body of at most 1 MB into dst. Unknown
// fields are ignored. It returns false after writing a 400 response.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(dst); err != nil {
		WriteError(w, http.StatusBadRequest, CodeInvalidJSON, "Request body must be valid JSON of at most 1 MB.")
		return false
	}
	// Reject trailing values (e.g. two concatenated JSON documents).
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		WriteError(w, http.StatusBadRequest, CodeInvalidJSON, "Request body must contain a single JSON value.")
		return false
	}
	return true
}

// QueryInt returns the integer value of a query parameter, or def when the
// parameter is absent or malformed.
func QueryInt(r *http.Request, key string, def int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return def
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return value
}

// Pagination holds validated list-query parameters.
type Pagination struct {
	Limit  int
	Cursor string
}

// ParsePagination reads ?limit=&cursor=, clamps the limit to [1, MaxLimit] and
// discards unreasonably long cursors.
func ParsePagination(r *http.Request) Pagination {
	limit := QueryInt(r, "limit", DefaultLimit)
	if limit < 1 || limit > MaxLimit {
		limit = DefaultLimit
	}
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	if len(cursor) > MaxCursorLen {
		cursor = ""
	}
	return Pagination{Limit: limit, Cursor: cursor}
}

// EncodeCursor encodes an opaque cursor value for a response.
func EncodeCursor(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

// DecodeCursor decodes a cursor produced by EncodeCursor.
func DecodeCursor(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	if len(encoded) > MaxCursorLen {
		return "", fmt.Errorf("invalid cursor")
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("invalid cursor")
	}
	if len(raw) > MaxCursorLen {
		return "", fmt.Errorf("invalid cursor")
	}
	return string(raw), nil
}

// EncodeIDCursor encodes an integer ID as a cursor.
func EncodeIDCursor(id int64) string {
	return EncodeCursor(strconv.FormatInt(id, 10))
}

// DecodeIDCursor decodes a cursor produced by EncodeIDCursor.
func DecodeIDCursor(encoded string) (int64, error) {
	raw, err := DecodeCursor(encoded)
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 0 {
		return 0, fmt.Errorf("invalid cursor")
	}
	return id, nil
}
