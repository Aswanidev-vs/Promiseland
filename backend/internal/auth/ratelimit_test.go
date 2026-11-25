package auth

import (
	"net/http/httptest"
	"testing"
)

func TestClientIPIgnoresForwardedHeadersFromPublicPeers(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	r.RemoteAddr = "203.0.113.9:4567"
	r.Header.Set("X-Forwarded-For", "10.0.0.1, 198.51.100.7")

	if got := clientIP(r); got != "203.0.113.9" {
		t.Fatalf("clientIP = %q, want the connection address", got)
	}
}

func TestClientIPUsesTheLastProxiedHop(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	r.RemoteAddr = "172.18.0.5:4567"
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 198.51.100.7")

	if got := clientIP(r); got != "198.51.100.7" {
		t.Fatalf("clientIP = %q, want the hop our proxy appended", got)
	}

	r.Header.Del("X-Forwarded-For")
	r.Header.Set("X-Real-IP", "198.51.100.8")
	if got := clientIP(r); got != "198.51.100.8" {
		t.Fatalf("clientIP = %q, want X-Real-IP", got)
	}
}

func TestClientIPFallsBackWhenForwardedHeadersAreMalformed(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	r.RemoteAddr = "127.0.0.1:4567"
	r.Header.Set("X-Forwarded-For", "not-an-ip")

	if got := clientIP(r); got != "127.0.0.1" {
		t.Fatalf("clientIP = %q, want the loopback peer", got)
	}
}
