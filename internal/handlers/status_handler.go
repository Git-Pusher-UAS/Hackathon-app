package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Pinger es lo mínimo que necesita el status para comprobar la base de datos.
// Lo implementa *sql.DB.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// ServiceName identifica el servicio en las respuestas de status.
const ServiceName = "backend-hackathon"

// StatusHandler responde al readiness probe con el estado de la base de datos.
type StatusHandler struct {
	db        Pinger
	version   string
	startedAt time.Time
}

// NewStatusHandler construye el handler de status.
func NewStatusHandler(db Pinger, version string) *StatusHandler {
	return &StatusHandler{
		db:        db,
		version:   version,
		startedAt: time.Now(),
	}
}

// Status comprueba la conexión a la base de datos y devuelve 200 o 503.
func (h *StatusHandler) Status(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
	defer cancel()

	start := time.Now()
	err := h.db.PingContext(ctx)
	latency := time.Since(start).Milliseconds()

	uptime := int64(time.Since(h.startedAt).Seconds())

	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status":         "degraded",
			"service":        ServiceName,
			"version":        h.version,
			"uptime_seconds": uptime,
			"database": fiber.Map{
				"status":     "down",
				"latency_ms": latency,
				"error":      err.Error(),
			},
		})
	}

	return c.JSON(fiber.Map{
		"status":         "ok",
		"service":        ServiceName,
		"version":        h.version,
		"uptime_seconds": uptime,
		"database": fiber.Map{
			"status":     "ok",
			"latency_ms": latency,
		},
	})
}
