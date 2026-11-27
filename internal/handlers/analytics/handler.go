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
