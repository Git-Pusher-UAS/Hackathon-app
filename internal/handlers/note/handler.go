// Package note expone POST /notes/sync.
package note

import (
	"errors"
	"log"

	hu "backend-hackathon/internal/handlers/handlerutil"
	"backend-hackathon/internal/httpx"
	nmodels "backend-hackathon/internal/models/note"
	svc "backend-hackathon/internal/services/note"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ nmodels.SyncRequest

// Sync sincroniza borradores de notas guardados sin conexión.
//
// @Summary      Sincronizar notas offline
// @Description  Hasta 20 borradores escritos sin conexión. Siempre responde 200 con un resultado por borrador, en el mismo orden: applied (se guardó o ya estaba), blocked con reason assignment_revoked o validation_failed (se descarta), conflict con reason record_closed o already_corrected (la persona decide) y retryable (error temporal; reintentar con el mismo operationId). Cada borrador lleva su propio operationId. baseVersionTag se acepta como referencia. Un 401 invalida todo el lote. Requiere casefile.write.
// @Tags         notes
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-CSRF-Token  header  string               true  "Token CSRF de la sesión"
// @Param        request       body    nmodels.SyncRequest  true  "Lote de notas"
// @Success      200  {object}  nmodels.SyncResponse
// @Failure      400  {object}  httpx.ErrorResponse
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Router       /notes/sync [post]
func (h *Handler) Sync(c *fiber.Ctx) error {
	actor, err := hu.Actor(c)
	if err != nil {
		return err
	}
	var req nmodels.SyncRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.Sync(c.Context(), actor, req)
	if err != nil {
		var ve *svc.ValidationError
		if errors.As(err, &ve) {
			return httpx.E(fiber.StatusUnprocessableEntity, "validation_failed", ve.Msg)
		}
		log.Printf("note: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "internal_error", "error interno")
	}
	return c.JSON(res)
}
