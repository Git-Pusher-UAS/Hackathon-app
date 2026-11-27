package auditlog

import (
	"errors"
	"log"
	"strconv"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	almodels "backend-hackathon/internal/models/auditlog"
	repository "backend-hackathon/internal/repository/repositories"
	svc "backend-hackathon/internal/services/auditlog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ almodels.Entry

func (h *Handler) List(c *fiber.Ctx) error {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	q := svc.Query{Action: c.Query("action")}
	if raw := c.Query("actorId"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "actorId invalido")
		}
		q.ActorID = &id
	}
	if raw := c.Query("recordId"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "recordId invalido")
		}
		q.RecordID = &id
	}
	if raw := c.Query("from"); raw != "" {
		t, err := parseTime(raw)
		if err != nil {
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "from invalido")
		}
		q.Rango.Desde = t
	}
	if raw := c.Query("to"); raw != "" {
		t, err := parseTime(raw)
		if err != nil {
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "to invalido")
		}
		q.Rango.Hasta = t
	}
	limit := 100
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 200 {
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "limit invalido")
		}
		limit = n
	}
	res, err := h.service.Logs(c.Context(), actor, q, repository.PageRequest{Limit: limit, Cursor: c.Query("cursor")})
	if err != nil {
		var he *httpx.Error
		if errors.As(err, &he) {
			return he
		}
		log.Printf("auditlog: %v", err)
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
