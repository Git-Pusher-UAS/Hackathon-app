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

// Pending lista encuestas pendientes de la usuaria.
//
// @Summary      Listar encuestas pendientes
// @Description  Devuelve encuestas pendientes con títulos neutros.
// @Tags         surveys
// @Produce      json
// @Param        X-CSRF-Token  header  string  false  "Token CSRF portal"
// @Success      200  {array}   smodels.Encuesta
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /surveys/pending [get]
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

// Get devuelve una encuesta.
//
// @Summary      Obtener encuesta
// @Description  Devuelve una encuesta con preguntas.
// @Tags         surveys
// @Produce      json
// @Param        id  path  string  true  "ID de la encuesta"
// @Success      200  {object}  smodels.Encuesta
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      404  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /surveys/{id} [get]
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

// Submit envía respuestas de encuesta.
//
// @Summary      Enviar respuestas de encuesta
// @Description  Guarda respuestas desvinculadas de identidad; idempotente por operationId.
// @Tags         surveys
// @Accept       json
// @Produce      json
// @Param        id            path    string                 true  "ID de la encuesta"
// @Param        X-CSRF-Token  header  string                 true  "Token CSRF portal"
// @Param        request       body    smodels.SubmitRequest  true  "Respuestas y operationId"
// @Success      204  "Sin contenido"
// @Failure      400  {object}  httpx.ErrorResponse
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      404  {object}  httpx.ErrorResponse
// @Failure      409  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /surveys/{id}/responses [post]
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

// Results devuelve resultados agregados de encuestas.
//
// @Summary      Resultados de encuestas
// @Description  Devuelve agregados con supresión cuando respuestas < 5.
// @Tags         surveys
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   smodels.Resultado
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /surveys/results [get]
func (h *Handler) Results(c *fiber.Ctx) error {
	res, err := h.service.Results(c.Context())
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}
