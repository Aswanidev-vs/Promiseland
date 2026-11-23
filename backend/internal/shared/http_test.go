package shared

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus int
		wantBody   string
		wantEmpty  bool
	}{
		{
			name:       "object",
			value:      map[string]string{"status": "ok"},
			wantStatus: http.StatusOK,
			wantBody:   `{"status":"ok"}`,
		},
		{
			name:       "created",
			value:      map[string]any{"id": 7, "ok": true},
			wantStatus: http.StatusCreated,
			wantBody:   `{"id":7,"ok":true}`,
		},
		{
			name:       "nil writes no body",
			value:      nil,
			wantStatus: http.StatusNoContent,
			wantEmpty:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			WriteJSON(rec, tt.wantStatus, tt.value)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q", got)
			}
			body := strings.TrimSpace(rec.Body.String())
			if tt.wantEmpty {
				if body != "" {
					t.Errorf("body = %q, want empty", body)
				}
				return
			}
			if body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		message string
	}{
		{name: "not found", code: CodeNotFound, message: "Endpoint not found."},
		{name: "auth", code: CodeAuthRequired, message: "Sign in to continue."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			WriteError(rec, http.StatusUnauthorized, tt.code, tt.message)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d", rec.Code)
			}
			var payload struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
					Details any    `json:"details"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode error response: %v (%q)", err, rec.Body.String())
			}
			if payload.Error.Code != tt.code || payload.Error.Message != tt.message {
				t.Errorf("error = %+v", payload.Error)
			}
			if payload.Error.Details != nil {
				t.Errorf("details should be omitted, got %v", payload.Error.Details)
			}
		})
	}
}

func TestDecodeJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	tests := []struct {
		name    string
		body    string
		wantOK  bool
		wantErr string
	}{
		{name: "valid", body: `{"name":"Sakura","extra":1}`, wantOK: true},
		{name: "invalid", body: `{`, wantErr: CodeInvalidJSON},
		{name: "trailing value", body: `{"name":"a"}{"name":"b"}`, wantErr: CodeInvalidJSON},
		{name: "oversized", body: `{"name":"` + strings.Repeat("x", 1<<20) + `"}`, wantErr: CodeInvalidJSON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			var got payload
			ok := DecodeJSON(rec, req, &got)
			if ok != tt.wantOK {
				t.Fatalf("DecodeJSON = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				var envelope struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
					t.Fatalf("decode error body: %v", err)
				}
				if envelope.Error.Code != tt.wantErr {
					t.Errorf("code = %q, want %q", envelope.Error.Code, tt.wantErr)
				}
				return
			}
			if got.Name == "" {
				t.Errorf("decoded payload = %+v", got)
			}
		})
	}
}

func TestQueryIntAndPagination(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=3&limit=7&cursor=abc", nil)
	if got := QueryInt(req, "page", 1); got != 3 {
		t.Errorf("QueryInt = %d, want 3", got)
	}
	if got := QueryInt(req, "missing", 42); got != 42 {
		t.Errorf("QueryInt default = %d, want 42", got)
	}
	req = httptest.NewRequest(http.MethodGet, "/?limit=nope", nil)
	if got := QueryInt(req, "limit", DefaultLimit); got != DefaultLimit {
		t.Errorf("QueryInt malformed = %d, want %d", got, DefaultLimit)
	}

	p := ParsePagination(httptest.NewRequest(http.MethodGet, "/?limit=7&cursor=abc", nil))
	if p.Limit != 7 || p.Cursor != "abc" {
		t.Errorf("pagination = %+v", p)
	}
	p = ParsePagination(httptest.NewRequest(http.MethodGet, "/?limit=999&cursor="+strings.Repeat("x", MaxCursorLen+1), nil))
	if p.Limit != DefaultLimit || p.Cursor != "" {
		t.Errorf("clamped pagination = %+v", p)
	}
}

func TestCursorRoundTrip(t *testing.T) {
	for _, value := range []string{"42", "post:2026-11-23T10:00:00Z:17", ""} {
		encoded := EncodeCursor(value)
		got, err := DecodeCursor(encoded)
		if err != nil {
			t.Fatalf("DecodeCursor(%q): %v", encoded, err)
		}
		if got != value {
			t.Errorf("round trip = %q, want %q", got, value)
		}
	}

	encoded := EncodeIDCursor(12345)
	id, err := DecodeIDCursor(encoded)
	if err != nil || id != 12345 {
		t.Errorf("ID cursor = %d, %v", id, err)
	}
	if _, err := DecodeCursor("not-valid-base64!"); err == nil {
		t.Error("expected error for malformed cursor")
	}
}
