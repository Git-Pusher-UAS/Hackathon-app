package middleware

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"backend-hackathon/internal/httpx"

	"github.com/gofiber/fiber/v3"
)

func TestRequestID(t *testing.T) {
	seen := make(map[string]bool)
	for _, supplied := range []string{"", "", "client-id", "identificador-é", "id\tcon-tab"} {
		app := fiber.New()
		app.Use(RequestID)
		var ctxID string
		app.Get("/", func(c fiber.Ctx) error {
			ctxID = RequestIDFrom(c.Context())
			return c.SendStatus(204)
		})
		req := httptest.NewRequest("GET", "/", nil)
		if supplied != "" {
			req.Header.Set(HeaderRequestID, supplied)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		id := resp.Header.Get(HeaderRequestID)
		if id != ctxID {
			t.Errorf("context ID %q != header %q", ctxID, id)
		}
		if supplied != "" {
			if id != supplied {
				t.Errorf("supplied ID changed: %q -> %q", supplied, id)
			}
		} else if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(id) || seen[id] {
			t.Errorf("invalid generated ID: %q", id)
		}
		seen[id] = true
	}
	if RequestIDFrom(context.Background()) != "" {
		t.Error("unexpected ID on empty context")
	}
}

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	oldWriter, oldFlags, oldPrefix := log.Writer(), log.Flags(), log.Prefix()
	log.SetOutput(&buf)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() { log.SetOutput(oldWriter); log.SetFlags(oldFlags); log.SetPrefix(oldPrefix) })
	return &buf
}

func TestLogging(t *testing.T) {
	buf := captureLogs(t)
	app := fiber.New()
	app.Use(RequestID, Logging())
	app.Get("/health", func(c fiber.Ctx) error { return httpx.JSON(c, 200, map[string]string{"status": "ok"}) })
	req := httptest.NewRequest("GET", "/health?a=1", nil)
	req.Header.Set(HeaderRequestID, "log-id")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if !regexp.MustCompile(`^GET /health\?a=1 200 16B [^ ]+ request_id=log-id\n$`).MatchString(buf.String()) {
		t.Errorf("log format changed: %q", buf.String())
	}
}

func TestRecover(t *testing.T) {
	buf := captureLogs(t)
	for _, value := range []any{"secret-panic", fiber.ErrNotFound} {
		app := fiber.New()
		app.Use(Recover(), RequestID, Logging(), CORS("*"))
		app.Get("/", func(c fiber.Ctx) error { panic(value) })
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set(HeaderRequestID, "panic-id")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 500 || string(body) != "{\"error\":\"internal_error\",\"message\":\"error interno del servidor\"}\n" {
			t.Errorf("panic response: %d %s", resp.StatusCode, body)
		}
		if resp.Header.Get("Access-Control-Allow-Origin") != "*" || resp.Header.Get(HeaderRequestID) != "panic-id" {
			t.Errorf("lost headers: %v", resp.Header)
		}
	}
	if !strings.Contains(buf.String(), "panic en GET / (request_id=panic-id)") || !strings.Contains(buf.String(), "runtime/debug.Stack") {
		t.Errorf("missing panic details: %s", buf.String())
	}
}
