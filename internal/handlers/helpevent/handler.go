// Package helpevent expone POST /api/v1/help-events.
package helpevent

import (
	"errors"
	"log"

	hu "backend-hackathon/internal/handlers/handlerutil"
	"backend-hackathon/internal/httpx"
	hemodels "backend-hackathon/internal/models/helpevent"
	svc "backend-hackathon/internal/services/helpevent"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ hemodels.CreateRequest

// Create registra que el personal abrió el marcador hacia un contacto de ayuda.
//
// @Summary      Registrar evento de ayuda
// @Description  El personal abrió el marcador telefónico hacia un contacto del directorio. Con Idempotency-Key: repetir la clave no duplica el evento. Si viene operationId en el cuerpo, debe ser igual a la clave.
// @Tags         help-events
// @Accept       json
// @Security     BearerAuth
// @Param        X-CSRF-Token     header  string                  true  "Token CSRF de la sesión"
// @Param        Idempotency-Key  header  string                  true  "UUID de la operación"
// @Param        request          body    hemodels.CreateRequest  true  "Evento"
// @Success      204  "Registrado"
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      409  {object}  httpx.ErrorResponse  "idempotency_mismatch"
// @Failure      422  {object}  httpx.ErrorResponse  "validation_failed con fields"
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
		var he *httpx.Error
		if errors.As(err, &he) {
			return he
		}
		log.Printf("helpevent: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "internal_error", "error interno")
	}
	return c.SendStatus(fiber.StatusNoContent)
}
