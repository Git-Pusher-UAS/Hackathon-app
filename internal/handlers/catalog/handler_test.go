package catalog

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	catmodels "backend-hackathon/internal/models/catalog"
	svc "backend-hackathon/internal/services/catalog"

	"github.com/gofiber/fiber/v2"
)

type fakeService struct{}

func (fakeService) Catalogs(context.Context) (*catmodels.Response, error) {
	return &catmodels.Response{
		Departamentos:     []catmodels.Departamento{{Codigo: "trabajo_social", Nombre: "Trabajo social"}},
		MotivosRevelacion: svc.MotivosRevelacion,
		Vacantes:          svc.VocabularioVacantes,
	}, nil
}

func TestCatalogsETag(t *testing.T) {
	app := fiber.New()
	app.Get("/catalogs", New(fakeService{}).Catalogs)

	resp, err := app.Test(httptest.NewRequest("GET", "/catalogs", nil))
	if err != nil {
		t.Fatal(err)
	}
	etag := resp.Header.Get("ETag")
	if resp.StatusCode != 200 || etag == "" || resp.Header.Get("Cache-Control") != "private, max-age=3600" {
		t.Fatalf("status=%d etag=%q cache=%q", resp.StatusCode, etag, resp.Header.Get("Cache-Control"))
	}
	raw, _ := io.ReadAll(resp.Body)
	var body catmodels.Response
	if err := json.Unmarshal(raw, &body); err != nil || len(body.Vacantes.Habilidades) != 12 || body.Departamentos[0].Codigo != "trabajo_social" {
		t.Fatalf("cuerpo: %s", raw)
	}

	req := httptest.NewRequest("GET", "/catalogs", nil)
	req.Header.Set("If-None-Match", etag)
	resp2, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp2.StatusCode != 304 {
		t.Fatalf("If-None-Match: status=%d", resp2.StatusCode)
	}
}
