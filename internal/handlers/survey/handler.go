package survey

import (
	"errors"
	"log"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	smodels "backend-hackathon/internal/models/survey"
	svc "backend-hackathon/internal/services/survey"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

func mapError(err error) error {
	var he *httpx.Error
	switch {
	case errors.As(err, &he):
		return he
	case errors.Is(err, svc.ErrNotFound):
		return httpx.E(fiber.StatusNotFound, "RESOURCE_NOT_FOUND", "recurso no disponible")
	case errors.Is(err, svc.ErrConflict):
		return httpx.E(fiber.StatusConflict, "CONFLICT", "encuesta ya contestada")
	default:
		log.Printf("survey: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
	}
}

func (h *Handler) Pending(c *fiber.Ctx) error {
	cuentaID, ok := middleware.PortalCuentaID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	res, err := h.service.Pending(c.Context(), cuentaID)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}

func (h *Handler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "id invalido")
	}
	res, err := h.service.Get(c.Context(), id)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}

func (h *Handler) Submit(c *fiber.Ctx) error {
	cuentaID, ok := middleware.PortalCuentaID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "id invalido")
	}
	var req smodels.SubmitRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	if err := h.service.Submit(c.Context(), cuentaID, id, req); err != nil {
		return mapError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) Results(c *fiber.Ctx) error {
	res, err := h.service.Results(c.Context())
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}
