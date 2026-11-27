// Package helpevent expone POST /api/help-events.
package helpevent

import (
	"errors"
	"log"
	"strings"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	hemodels "backend-hackathon/internal/models/helpevent"
	svc "backend-hackathon/internal/services/helpevent"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ hemodels.CreateRequest

// Create registra un evento de ayuda.
//
// @Summary      Registrar evento de ayuda
// @Description  Registra que el personal abrio el marcador hacia un contacto. Idempotente por operationId.
// @Tags         help-events
// @Accept       json
// @Security     BearerAuth
// @Param        X-CSRF-Token  header  string                 true  "Token CSRF"
// @Param        request       body    hemodels.CreateRequest  true  "Evento"
// @Success      204  "Registrado"
// @Failure      400  {object}  httpx.ErrorResponse
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      409  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /help-events [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	var req hemodels.CreateRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	opRaw := strings.TrimSpace(req.OperationID)
	if opRaw == "" {
		opRaw = strings.TrimSpace(c.Get("Idempotency-Key"))
	}
	if opRaw == "" {
		opRaw = strings.TrimSpace(c.Get("X-Operation-Id"))
	}
	opID, err := uuid.Parse(opRaw)
	if err != nil {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "operationId invalido")
	}
	if err := h.service.Record(c.Context(), actor, opID, req); err != nil {
		var he *httpx.Error
		if errors.As(err, &he) {
			return he
		}
		log.Printf("helpevent: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
	}
	return c.SendStatus(fiber.StatusNoContent)
}
