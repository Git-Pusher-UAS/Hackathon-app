package analytics

import (
	"errors"
	"log"
	"time"

	"backend-hackathon/internal/httpx"
	anmodels "backend-hackathon/internal/models/analytics"
	svc "backend-hackathon/internal/services/analytics"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ anmodels.Summary

// Summary devuelve indicadores agregados con supresión.
//
// @Summary      Resumen analítico
// @Description  Devuelve indicadores agregados con supresión k<5.
// @Tags         analytics
// @Produce      json
// @Security     BearerAuth
// @Param        period  query  string  false  "30d | 6m | 12m"
// @Param        from    query  string  false  "Fecha/hora inicial"
// @Param        to      query  string  false  "Fecha/hora final"
// @Param        type    query  string  false  "Tipo de caso (UUID)"
// @Param        status  query  string  false  "Estado del expediente"
// @Success      200  {object}  anmodels.Summary
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /analytics/summary [get]
func (h *Handler) Summary(c *fiber.Ctx) error {
	q := svc.SummaryQuery{Period: c.Query("period"), Type: c.Query("type"), Status: c.Query("status")}
	if raw := c.Query("from"); raw != "" {
		t, err := parseTime(raw)
		if err != nil {
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "from invalido")
		}
		q.From = &t
	}
	if raw := c.Query("to"); raw != "" {
		t, err := parseTime(raw)
		if err != nil {
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "to invalido")
		}
		q.To = &t
	}
	res, err := h.service.Summary(c.Context(), q)
	if err != nil {
		var he *httpx.Error
		if errors.As(err, &he) {
			return he
		}
		log.Printf("analytics: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
	}
	return c.JSON(res)
}

// Heatmap devuelve el mapa de calor agregado.
//
// @Summary      Mapa de calor analítico
// @Description  Devuelve celdas por sector con supresión k<5.
// @Tags         analytics
// @Produce      json
// @Security     BearerAuth
// @Param        from  query  string  false  "Fecha/hora inicial"
// @Param        to    query  string  false  "Fecha/hora final"
// @Param        type  query  string  false  "Tipo de caso (UUID)"
// @Success      200  {array}   anmodels.HeatmapCell
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /analytics/heatmap [get]
func (h *Handler) Heatmap(c *fiber.Ctx) error {
	q := svc.HeatmapQuery{Type: c.Query("type")}
	if raw := c.Query("from"); raw != "" {
		t, err := parseTime(raw)
		if err != nil {
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "from invalido")
		}
		q.From = &t
	}
	if raw := c.Query("to"); raw != "" {
		t, err := parseTime(raw)
		if err != nil {
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "to invalido")
		}
		q.To = &t
	}
	res, err := h.service.Heatmap(c.Context(), q)
	if err != nil {
		var he *httpx.Error
		if errors.As(err, &he) {
			return he
		}
		log.Printf("analytics: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
	}
	return c.JSON(res)
}

func parseTime(raw string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02T15:04:05", raw)
}
