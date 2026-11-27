// Package helpevent expone POST /help-events.
package helpevent

import (
	"errors"
	"log"

	hu "backend-hackathon/internal/handlers/handlerutil"
	"backend-hackathon/internal/httpx"
	hemodels "backend-hackathon/internal/models/helpevent"
	repository "backend-hackathon/internal/repository/repositories"
	svc "backend-hackathon/internal/services/helpevent"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ hemodels.CreateRequest

// Create registra la apertura del marcador hacia un contacto de ayuda.
//
// @Summary      Registrar evento de ayuda
// @Description  Registra que el personal abrió el marcador telefónico. Idempotente con X-Operation-Id: un reintento con la misma clave responde 204 sin duplicar. No guarda a qué contacto se llamó. API-040.
// @Tags         help-events
// @Accept       json
// @Security     BearerAuth
// @Param        X-CSRF-Token    header  string                    true  "Token CSRF de la sesión"
// @Param        X-Operation-Id  header  string                    true  "UUID de operación idempotente"
// @Param        request         body    hemodels.CreateRequest    true  "Evento"
// @Success      204  "Registrado"
// @Failure      400  {object}  httpx.ErrorResponse
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      409  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /help-events [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	actor, err := hu.Actor(c)
	if err != nil {
		return err
	}
	op, err := hu.OperationID(c)
	if err != nil {
		return err
	}
	var req hemodels.CreateRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	if err := h.service.Record(c.Context(), actor, op, req); err != nil {
		var inv *svc.ErrInvalid
		switch {
		case errors.As(err, &inv):
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", inv.Msg)
		case errors.Is(err, repository.ErrConflict):
			return httpx.E(fiber.StatusConflict, "IDEMPOTENCY_CONFLICT", "X-Operation-Id ya usado")
		default:
			log.Printf("helpevent: %v", err)
			return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
		}
	}
	return c.SendStatus(fiber.StatusNoContent)
}
