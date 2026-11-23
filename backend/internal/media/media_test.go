package media

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 3))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestUploadServeAndTraversal(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MEDIA_PATH", root)
	d, user := testDeps(t)
	h := testRouter(d)
	cookie := authCookie(t, d, user)
	data := pngBytes(t)

	rec := uploadRequest(t, h, "outfit.png", "image/png", data, cookie)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body.String())
	}
	uploaded := decodeMedia(t, rec.Body.Bytes())
	if !strings.HasPrefix(uploaded.URL, "/media/") || uploaded.MimeType != "image/png" ||
		uploaded.Width != 4 || uploaded.Height != 3 || uploaded.SizeBytes != int64(len(data)) {
		t.Fatalf("upload result = %+v", uploaded)
	}
	key := strings.TrimPrefix(uploaded.URL, "/media/")
	if len(key) != 52 || !strings.HasSuffix(key, ".png") {
		t.Fatalf("storage key = %q", key)
	}
	if _, err := filepath.Abs(filepath.Join(root, key)); err != nil {
		t.Fatal(err)
	}

	served := httptestGet(t, h, uploaded.URL, nil)
	if served.Code != http.StatusOK || !bytes.Equal(served.Body.Bytes(), data) {
		t.Fatalf("serve: %d %q", served.Code, served.Body.String())
	}
	if served.Header().Get("Content-Type") != "image/png" ||
		served.Header().Get("X-Content-Type-Options") != "nosniff" ||
		!strings.Contains(served.Header().Get("Cache-Control"), "max-age") {
		t.Fatalf("serve headers: %+v", served.Header())
	}

	for _, path := range []string{"/media/../secret", "/media/%2e%2e%2fsecret", "/media/not-a-key.png"} {
		if rec := httptestGet(t, h, path, nil); rec.Code != http.StatusNotFound {
			t.Fatalf("traversal %q: %d", path, rec.Code)
		}
	}
}

func TestUploadValidation(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MEDIA_PATH", root)
	d, user := testDeps(t)
	h := testRouter(d)
	cookie := authCookie(t, d, user)

	tests := []struct {
		name        string
		filename    string
		contentType string
		data        []byte
	}{
		{"bad declared mime", "photo.png", "text/html", []byte("<html></html>")},
		{"bad extension", "photo.svg", "image/svg+xml", []byte("<svg></svg>")},
		{"bytes do not match", "photo.png", "image/png", []byte("not an image")},
		{"too large", "photo.png", "image/png", append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte("x"), maxUploadBytes)...)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := uploadRequest(t, h, tt.filename, tt.contentType, tt.data, cookie)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d %s", rec.Code, rec.Body.String())
			}
			var envelope struct {
				Error struct {
					Code    string            `json:"code"`
					Details map[string]string `json:"details"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil ||
				envelope.Error.Code != "invalid_file" || envelope.Error.Details["file"] == "" {
				t.Fatalf("error = %s", rec.Body.String())
			}
		})
	}

	var count int
	if err := d.DB.QueryRow(`SELECT COUNT(*) FROM media`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("failed uploads left %d rows, err %v", count, err)
	}
	entries, err := filepath.Glob(filepath.Join(root, "*"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed uploads left files: %v, %v", entries, err)
	}
}

func TestUnauthenticatedUploadRejected(t *testing.T) {
	t.Setenv("MEDIA_PATH", t.TempDir())
	d, _ := testDeps(t)
	h := testRouter(d)
	if rec := uploadRequest(t, h, "a.png", "image/png", pngBytes(t), nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated upload: %d", rec.Code)
	}
}

func decodeMedia(t *testing.T, raw []byte) Media {
	t.Helper()
	var media Media
	if err := json.Unmarshal(raw, &media); err != nil {
		t.Fatal(err)
	}
	return media
}

func httptestGet(t *testing.T, h http.Handler, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
