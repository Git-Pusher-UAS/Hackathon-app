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

// List lista vacantes.
//
// @Summary      Listar vacantes
// @Description  Lista vacantes con filtros y marca postulación si hay sesión portal.
// @Tags         jobs
// @Produce      json
// @Param        sector    query  string  false  "Sector"
// @Param        schedule  query  string  false  "Horario"
// @Param        type      query  string  false  "Tipo de empleo"
// @Param        location  query  string  false  "Ubicación"
// @Success      200  {array}   jmodels.Job
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /jobs [get]
func (h *Handler) List(c *fiber.Ctx) error {
	f := repository.VacanteFilter{Sector: c.Query("sector"), Horario: c.Query("schedule"), TipoEmpleo: c.Query("type"), Ubicacion: c.Query("location")}
	res, err := h.service.List(c.Context(), f, optionalPortalID(c))
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}

// Get obtiene una vacante.
//
// @Summary      Obtener vacante
// @Description  Devuelve el detalle de una vacante.
// @Tags         jobs
// @Produce      json
// @Param        id  path  string  true  "ID de la vacante"
// @Success      200  {object}  jmodels.Job
// @Failure      404  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /jobs/{id} [get]
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

// Create crea una vacante.
//
// @Summary      Crear vacante
// @Description  Crea una vacante generando metadatos en servidor.
// @Tags         jobs
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-CSRF-Token  header  string                true  "Token CSRF"
// @Param        request       body    jmodels.CreateRequest  true  "Datos de la vacante"
// @Success      201  {object}  jmodels.Job
// @Failure      400  {object}  httpx.ErrorResponse
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /jobs [post]
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

// Apply postula a la usuaria portal a una vacante.
//
// @Summary      Postularse a vacante
// @Description  Registra postulación de la usuaria autenticada del portal.
// @Tags         jobs
// @Produce      json
// @Param        id            path    string  true  "ID de la vacante"
// @Param        X-CSRF-Token  header  string  true  "Token CSRF portal"
// @Success      204  "Sin contenido"
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      404  {object}  httpx.ErrorResponse
// @Failure      409  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /jobs/{id}/applications [post]
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
