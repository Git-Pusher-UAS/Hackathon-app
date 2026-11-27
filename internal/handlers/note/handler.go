package note

import (
	"errors"
	"log"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	nmodels "backend-hackathon/internal/models/note"
	svc "backend-hackathon/internal/services/note"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ nmodels.SyncResult

// Sync sincroniza borradores de notas offline.
//
// @Summary      Sincronizar notas offline
// @Description  Procesa un lote idempotente de notas (1..20) y devuelve resultado por nota.
// @Tags         notes
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-CSRF-Token  header  string               true  "Token CSRF"
// @Param        request       body    nmodels.SyncRequest  true  "Lote de notas"
// @Success      200  {array}   nmodels.SyncResult
// @Failure      400  {object}  httpx.ErrorResponse
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /notes/sync [post]
func (h *Handler) Sync(c *fiber.Ctx) error {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	var req nmodels.SyncRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.Sync(c.Context(), actor, req)
	if err != nil {
		var ve *svc.ValidationError
		if errors.As(err, &ve) {
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", ve.Msg)
		}
		log.Printf("note: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
	}
	return c.JSON(res)
}
