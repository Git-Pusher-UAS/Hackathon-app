// Package analytics expone /api/v1/analytics.
package analytics

import (
	"errors"
	"log"

	"backend-hackathon/internal/httpx"
	anmodels "backend-hackathon/internal/models/analytics"
	svc "backend-hackathon/internal/services/analytics"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ anmodels.Summary

func mapError(err error) error {
	var he *httpx.Error
	if errors.As(err, &he) {
		return he
	}
	log.Printf("analytics: %v", err)
	return httpx.E(fiber.StatusInternalServerError, "internal_error", "error interno")
}

// Summary devuelve los indicadores agregados del tablero.
//
// @Summary      Indicadores del tablero
// @Description  Expedientes abiertos y canalizaciones del periodo (con su cambio frente al periodo anterior), por mes, por origen de contacto y por área. El corte es la medianoche de hoy en Los Mochis. Las cifras menores a 5 llegan en null con suprimido=true. Requiere analytics.read.
// @Tags         analytics
// @Produce      json
// @Security     BearerAuth
// @Param        period  query  string  false  "30d (por defecto) | 6m | 12m"
// @Success      200  {object}  anmodels.Summary
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Router       /analytics/summary [get]
func (h *Handler) Summary(c *fiber.Ctx) error {
	res, err := h.service.Summary(c.Context(), c.Query("period"))
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}
