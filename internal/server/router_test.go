package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend-hackathon/internal/config"
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

type pingerFunc func(context.Context) error

func (f pingerFunc) PingContext(ctx context.Context) error { return f(ctx) }

func testConfig(origin string) *config.Config {
	return &config.Config{Server: config.ServerConfig{Port: "0", CORSOrigin: origin}}
}

func response(t *testing.T, app *fiber.App, method, path, origin string) (*http.Response, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Origin", origin)
	req.Header.Set(middleware.HeaderRequestID, "contract-test")
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	return resp, string(body)
}

func TestRouterContract(t *testing.T) {
	app := router(testConfig("http://localhost:3000"), pingerFunc(func(context.Context) error { return nil }), "test")
	for _, tc := range []struct {
		method, path string
		code         int
		body         string
	}{
		{"GET", "/health", 200, "{\"status\":\"ok\"}\n"},
		{"GET", "/health?test=1", 200, "{\"status\":\"ok\"}\n"},
		{"HEAD", "/health", 200, ""},
		{"HEAD", "/api/v1/status", 200, ""},
		{"POST", "/health", 405, "{\"error\":\"method_not_allowed\",\"message\":\"método no permitido\"}\n"},
		{"PUT", "/api/v1/status", 405, "{\"error\":\"method_not_allowed\",\"message\":\"método no permitido\"}\n"},
		{"DELETE", "/api/v1/status", 405, "{\"error\":\"method_not_allowed\",\"message\":\"método no permitido\"}\n"},
		{"PATCH", "/health", 405, "{\"error\":\"method_not_allowed\",\"message\":\"método no permitido\"}\n"},
		{"TRACE", "/health", 405, "{\"error\":\"method_not_allowed\",\"message\":\"método no permitido\"}\n"},
		{"PROPFIND", "/health", 405, "{\"error\":\"method_not_allowed\",\"message\":\"método no permitido\"}\n"},
		{"GET", "/missing", 404, "{\"error\":\"not_found\",\"message\":\"ruta no encontrada\"}\n"},
		{"POST", "/missing", 404, "{\"error\":\"not_found\",\"message\":\"ruta no encontrada\"}\n"},
		{"PROPFIND", "/missing", 404, "{\"error\":\"not_found\",\"message\":\"ruta no encontrada\"}\n"},
		{"GET", "/HEALTH", 404, "{\"error\":\"not_found\",\"message\":\"ruta no encontrada\"}\n"},
		{"GET", "/health/", 404, "{\"error\":\"not_found\",\"message\":\"ruta no encontrada\"}\n"},
		{"GET", "/api/v1/status/", 404, "{\"error\":\"not_found\",\"message\":\"ruta no encontrada\"}\n"},
		{"GET", "/api%2fv1/status", 404, "{\"error\":\"not_found\",\"message\":\"ruta no encontrada\"}\n"},
		{"GET", "/foo/%2e%2e/health", 404, "{\"error\":\"not_found\",\"message\":\"ruta no encontrada\"}\n"},
		{"GET", "/%68ealth", 200, "{\"status\":\"ok\"}\n"},
		{"OPTIONS", "/health", 204, ""},
		{"OPTIONS", "/missing", 204, ""},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			resp, body := response(t, app, tc.method, tc.path, "http://localhost:3000")
			if resp.StatusCode != tc.code || body != tc.body {
				t.Errorf("got %d %q; want %d %q", resp.StatusCode, body, tc.code, tc.body)
			}
			if tc.code != 204 && resp.Header.Get("Content-Type") != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q", resp.Header.Get("Content-Type"))
			}
			if resp.Header.Get(middleware.HeaderRequestID) != "contract-test" {
				t.Errorf("request ID = %q", resp.Header.Get(middleware.HeaderRequestID))
			}
			if tc.code == 405 && resp.Header.Get("Allow") != "GET, OPTIONS" {
				t.Errorf("Allow = %q", resp.Header.Get("Allow"))
			}
			if resp.Header.Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
				t.Errorf("missing CORS headers: %v", resp.Header)
			}
		})
	}
}

func TestCORSContract(t *testing.T) {
	for _, tc := range []struct{ allowed, origin, want, vary string }{
		{"*", "", "*", ""},
		{"", "http://example.com", "*", ""},
		{"  ", "", "*", ""},
		{"http://one.test, http://two.test", "http://two.test", "http://two.test", "Origin"},
		{"http://one.test", "http://other.test", "", ""},
		{"http://one.test", "", "", ""},
		{"http://one.test", "http://ONE.test", "", ""},
		{"*, http://one.test", "http://other.test", "", ""},
	} {
		app := router(testConfig(tc.allowed), pingerFunc(func(context.Context) error { t.Fatal("unexpected ping"); return nil }), "test")
		for _, method := range []string{"GET", "OPTIONS"} {
			resp, _ := response(t, app, method, "/health", tc.origin)
			wantCode := 200
			if method == "OPTIONS" {
				wantCode = 204
			}
			if resp.StatusCode != wantCode {
				t.Errorf("status = %d", resp.StatusCode)
			}
			for key, want := range map[string]string{
				"Access-Control-Allow-Origin":   tc.want,
				"Vary":                          tc.vary,
				"Access-Control-Allow-Methods":  "GET, POST, PUT, PATCH, DELETE, OPTIONS",
				"Access-Control-Allow-Headers":  "Content-Type, Authorization, If-Match, X-Request-ID",
				"Access-Control-Expose-Headers": "X-Request-ID",
				"Access-Control-Max-Age":        "600",
			} {
				if got := resp.Header.Get(key); got != want {
					t.Errorf("%s/%q: %s=%q; want %q", method, tc.allowed, key, got, want)
				}
			}
		}
	}
}

func TestStatusContract(t *testing.T) {
	for _, down := range []bool{false, true} {
		var pingCtx context.Context
		app := router(testConfig("*"), pingerFunc(func(ctx context.Context) error {
			pingCtx = ctx
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > 2*time.Second {
				t.Error("missing 2s ping deadline")
			}
			if middleware.RequestIDFrom(ctx) != "contract-test" {
				t.Error("missing request ID in ping context")
			}
			if down {
				return errors.New("database unavailable")
			}
			return nil
		}), "test-version")
		resp, body := response(t, app, "GET", "/api/v1/status", "")
		var data struct {
			Status   string `json:"status"`
			Service  string `json:"service"`
			Version  string `json:"version"`
			Uptime   int64  `json:"uptime_seconds"`
			Database struct {
				Status string `json:"status"`
				Error  string `json:"error"`
			} `json:"database"`
		}
		if err := json.Unmarshal([]byte(body), &data); err != nil {
			t.Fatal(err)
		}
		if data.Service != "backend-hackathon" || data.Version != "test-version" || data.Uptime < 0 {
			t.Errorf("invalid status: %s", body)
		}
		if down {
			if resp.StatusCode != 503 || data.Status != "degraded" || data.Database.Status != "down" || data.Database.Error != "database unavailable" {
				t.Errorf("invalid down response: %s", body)
			}
		} else if resp.StatusCode != 200 || data.Status != "ok" || data.Database.Status != "ok" || data.Database.Error != "" {
			t.Errorf("invalid healthy response: %s", body)
		}
		if pingCtx.Err() != context.Canceled {
			t.Errorf("ping context not released: %v", pingCtx.Err())
		}
	}
}

func TestStatusDeadline(t *testing.T) {
	app := router(testConfig("*"), pingerFunc(func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }), "test")
	resp, body := response(t, app, "GET", "/api/v1/status", "")
	if resp.StatusCode != 503 || !strings.Contains(body, "context deadline exceeded") {
		t.Errorf("got %d %s", resp.StatusCode, body)
	}
}

func TestCanonicalRedirect(t *testing.T) {
	app := router(testConfig("*"), pingerFunc(func(context.Context) error { return nil }), "test")
	for _, path := range []string{"//health", "/foo/../health", "/./health"} {
		resp, body := response(t, app, "GET", path+"?a=1&b=2", "")
		if resp.StatusCode != 307 || resp.Header.Get("Location") != "/health?a=1&b=2" || body != "<a href=\"/health?a=1&amp;b=2\">Temporary Redirect</a>.\n\n" {
			t.Errorf("%s: got %d %v %q", path, resp.StatusCode, resp.Header, body)
		}
	}
}

func TestLargeIgnoredRequestBody(t *testing.T) {
	app := router(testConfig("*"), pingerFunc(func(context.Context) error { return nil }), "test")
	req := httptest.NewRequest("POST", "/health", strings.NewReader(strings.Repeat("a", 5<<20)))
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 405 {
		t.Errorf("large body changed status: %d", resp.StatusCode)
	}
}
