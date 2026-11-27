package auditalert

import (
	"errors"
	"log"
	"strconv"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	"backend-hackathon/internal/models"
	aamodels "backend-hackathon/internal/models/auditalert"
	repository "backend-hackathon/internal/repository/repositories"
	svc "backend-hackathon/internal/services/auditalert"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ aamodels.Item

func (h *Handler) List(c *fiber.Ctx) error {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	var estado *models.EstadoAlerta
	if raw := c.Query("status"); raw != "" {
		st := models.EstadoAlerta(raw)
		switch st {
		case models.AlertaAbierta, models.AlertaRevisada:
			estado = &st
		default:
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "status invalido")
		}
	}
	limit := 100
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 200 {
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "limit invalido")
		}
		limit = n
	}
	res, err := h.service.List(c.Context(), actor, estado, repository.PageRequest{Limit: limit, Cursor: c.Query("cursor")})
	if err != nil {
		var he *httpx.Error
		if errors.As(err, &he) {
			return he
		}
		log.Printf("auditalert: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
	}
	return c.JSON(res)
}
