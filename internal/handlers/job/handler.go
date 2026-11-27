package job

import (
	"errors"
	"log"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	jmodels "backend-hackathon/internal/models/job"
	repository "backend-hackathon/internal/repository/repositories"
	svc "backend-hackathon/internal/services/job"

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
	case errors.Is(err, svc.ErrDuplicate):
		return httpx.E(fiber.StatusConflict, "CONFLICT", "ya postulada")
	default:
		log.Printf("job: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
	}
}

func optionalPortalID(c *fiber.Ctx) *uuid.UUID {
	id, ok := middleware.PortalCuentaID(c)
	if !ok {
		return nil
	}
	return &id
}

func (h *Handler) List(c *fiber.Ctx) error {
	f := repository.VacanteFilter{Sector: c.Query("sector"), Horario: c.Query("schedule"), TipoEmpleo: c.Query("type"), Ubicacion: c.Query("location")}
	res, err := h.service.List(c.Context(), f, optionalPortalID(c))
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
	res, err := h.service.Get(c.Context(), id, optionalPortalID(c))
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var req jmodels.CreateRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.Create(c.Context(), req)
	if err != nil {
		return mapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(res)
}

func (h *Handler) Apply(c *fiber.Ctx) error {
	cuentaID, ok := middleware.PortalCuentaID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "id invalido")
	}
	if err := h.service.Apply(c.Context(), id, cuentaID); err != nil {
		return mapError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
