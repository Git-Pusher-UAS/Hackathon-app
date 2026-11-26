package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// HealthResponse es la respuesta del liveness probe.
type HealthResponse struct {
	Status string `json:"status" example:"OK"`
	Time   string `json:"time" example:"2026-01-02T15:04:05Z"`
}

// Healthcheck es el liveness probe. No toca la base de datos.
//
// @Summary      Liveness probe
// @Description  Comprueba que el proceso está vivo. Se sirve en la raíz, fuera de /api.
// @Tags         health
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Router       /health [get]
func Healthcheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "OK",
		"time":   time.Now().Format(time.RFC3339),
	})
}
