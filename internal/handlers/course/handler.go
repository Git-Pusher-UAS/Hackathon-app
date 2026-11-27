package course

import (
	"errors"
	"log"
	"strconv"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	"backend-hackathon/internal/models"
	cmodels "backend-hackathon/internal/models/course"
	repository "backend-hackathon/internal/repository/repositories"
	svc "backend-hackathon/internal/services/course"

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
	case errors.Is(err, svc.ErrFull):
		return httpx.E(fiber.StatusConflict, "CONFLICT", "cupo lleno")
	case errors.Is(err, svc.ErrDuplicate):
		return httpx.E(fiber.StatusConflict, "CONFLICT", "ya inscrita")
	default:
		log.Printf("course: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
	}
}

func actorOf(c *fiber.Ctx) (repository.ActorContext, error) {
	a, ok := middleware.ActorFrom(c)
	if !ok {
		return a, httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	return a, nil
}

func (h *Handler) List(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	var tipo *models.TipoCurso
	if raw := c.Query("type"); raw != "" {
		t := models.TipoCurso(raw)
		if !svc.ValidTipo(t) {
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "type invalido")
		}
		tipo = &t
	}
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "limit invalido")
		}
		limit = n
	}
	res, err := h.service.List(c.Context(), actor, tipo, repository.PageRequest{Limit: limit, Cursor: c.Query("cursor")})
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

func (h *Handler) Create(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	var req cmodels.CreateRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.Create(c.Context(), actor, req)
	if err != nil {
		return mapError(err)
	}
	c.Set(fiber.HeaderLocation, "/api/courses/"+res.ID.String())
	return c.Status(fiber.StatusCreated).JSON(res)
}

func (h *Handler) Enroll(c *fiber.Ctx) error {
	cuentaID, ok := middleware.PortalCuentaID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "id invalido")
	}
	if err := h.service.Enroll(c.Context(), cuentaID, id); err != nil {
		return mapError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) Attendance(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "id invalido")
	}
	res, err := h.service.Attendance(c.Context(), id)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}

func (h *Handler) SetAttendance(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "id invalido")
	}
	var req []cmodels.MarcaAsistencia
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	if err := h.service.SetAttendance(c.Context(), actor, id, req); err != nil {
		return mapError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
