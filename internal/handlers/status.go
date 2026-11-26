package handlers

import (
	"context"
	"time"

	"backend-hackathon/internal/httpx"

	"github.com/gofiber/fiber/v3"
)

// Pinger es lo mínimo que necesita el status para comprobar la base de datos.
// Lo implementa *sql.DB.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// ServiceName identifica el servicio en las respuestas de status.
const ServiceName = "backend-hackathon"

// Status responde a los endpoints de salud.
type Status struct {
	db        Pinger
	version   string
	startedAt time.Time
}

// NewStatus construye el handler de status.
func NewStatus(db Pinger, version string) *Status {
	return &Status{
		db:        db,
		version:   version,
		startedAt: time.Now(),
	}
}

type databaseStatus struct {
	Status    string `json:"status"`
	LatencyMS int64  `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}

type statusResponse struct {
	Status        string         `json:"status"`
	Service       string         `json:"service"`
	Version       string         `json:"version"`
	UptimeSeconds int64          `json:"uptime_seconds"`
	Database      databaseStatus `json:"database"`
}

// Health es el liveness probe: no toca la base de datos.
func (h *Status) Health(c fiber.Ctx) error {
	return httpx.JSON(c, fiber.StatusOK, map[string]string{"status": "ok"})
}

// Status es el readiness probe: comprueba la conexión a la base de datos.
func (h *Status) Status(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
	defer cancel()

	start := time.Now()
	err := h.db.PingContext(ctx)
	latency := time.Since(start)

	resp := statusResponse{
		Status:        "ok",
		Service:       ServiceName,
		Version:       h.version,
		UptimeSeconds: int64(time.Since(h.startedAt).Seconds()),
		Database: databaseStatus{
			Status:    "ok",
			LatencyMS: latency.Milliseconds(),
		},
	}

	code := fiber.StatusOK
	if err != nil {
		resp.Status = "degraded"
		resp.Database.Status = "down"
		resp.Database.Error = err.Error()
		code = fiber.StatusServiceUnavailable
	}

	return httpx.JSON(c, code, resp)
}
