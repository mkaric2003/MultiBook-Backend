package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mkaric2003/multibook-backend/internal/platform/config"
)

func TestHealthz(t *testing.T) {
	handler := NewRouter(context.Background(), nil, nil, testConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestCORSPreflightForAllowedOrigin(t *testing.T) {
	handler := NewRouter(context.Background(), nil, nil, testConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodOptions, "/v1/users/me", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("allow origin = %q", got)
	}
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	handler := NewRouter(context.Background(), nil, nil, testConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("Origin", "https://untrusted.example")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func testConfig() config.Config {
	return config.Config{
		CORSAllowedOrigins: []string{"http://localhost:3000"},
		RateLimitRPS:       100,
		RateLimitBurst:     100,
	}
}

func TestSavedRoutesRequireAuthentication(t *testing.T) {
	for _, test := range []struct{ method, path string }{
		{http.MethodGet, "/v1/saved-businesses"},
		{http.MethodGet, "/v1/saved-businesses/11111111-1111-1111-1111-111111111111"},
		{http.MethodPut, "/v1/saved-businesses/11111111-1111-1111-1111-111111111111"},
		{http.MethodDelete, "/v1/saved-businesses/11111111-1111-1111-1111-111111111111"},
	} {
		t.Run(test.method+test.path, func(t *testing.T) {
			handler := NewRouter(context.Background(), nil, nil, testConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d", recorder.Code)
			}
		})
	}
}
