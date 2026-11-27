package analytics

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"backend-hackathon/internal/httpx"
	anmodels "backend-hackathon/internal/models/analytics"
	svc "backend-hackathon/internal/services/analytics"

	"github.com/gofiber/fiber/v2"
)

type stubService struct {
	summaryFn func(context.Context, svc.SummaryQuery) (*anmodels.Summary, error)
	heatmapFn func(context.Context, svc.HeatmapQuery) ([]anmodels.HeatmapCell, error)
}

func (s stubService) Summary(ctx context.Context, q svc.SummaryQuery) (*anmodels.Summary, error) {
	return s.summaryFn(ctx, q)
}
func (s stubService) Heatmap(ctx context.Context, q svc.HeatmapQuery) ([]anmodels.HeatmapCell, error) {
	return s.heatmapFn(ctx, q)
}

func TestSummarySuppression(t *testing.T) {
	h := New(stubService{summaryFn: func(context.Context, svc.SummaryQuery) (*anmodels.Summary, error) {
		return &anmodels.Summary{Corte: time.Now(), Metodologia: "m", Indicadores: []anmodels.Indicador{{Clave: "abiertos", Titulo: "Expedientes abiertos", Valor: nil, Suprimido: true}}, PorMes: []anmodels.Serie{}, PorOrigen: []anmodels.Serie{}, PorArea: []anmodels.Serie{}}, nil
	}})
	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error { return httpx.Write(c, err) }})
	app.Get("/api/analytics/summary", h.Summary)
	resp, err := app.Test(httptest.NewRequest("GET", "/api/analytics/summary?period=30d", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestHeatmapSuppressed(t *testing.T) {
	h := New(stubService{heatmapFn: func(context.Context, svc.HeatmapQuery) ([]anmodels.HeatmapCell, error) {
		return []anmodels.HeatmapCell{{SectorID: "s1", Count: nil, Suppressed: true}}, nil
	}})
	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error { return httpx.Write(c, err) }})
	app.Get("/api/analytics/heatmap", h.Heatmap)
	resp, err := app.Test(httptest.NewRequest("GET", "/api/analytics/heatmap?from=2026-01-01&to=2026-01-31&type=x", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
