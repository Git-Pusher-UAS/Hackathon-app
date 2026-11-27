package httpx

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestWriteIncludesRootCodeAndCorrelationID(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error { return Write(c, E(fiber.StatusConflict, "CONFLICT", "x")) })
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != "CONFLICT" {
		t.Fatalf("code=%v", body["code"])
	}
	if body["correlation_id"] == "" {
		t.Fatalf("missing correlation_id")
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["requestId"] != body["correlation_id"] {
		t.Fatalf("requestId mismatch")
	}
}
