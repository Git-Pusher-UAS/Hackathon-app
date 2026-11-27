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
// @Description  Lote de 1 a 20 notas. Cada nota lleva su propio operationId y se guarda en su propia transacción con las reglas de API-027. Devuelve un resultado por nota, en el mismo orden: applied, blocked (no reintentar), conflict o retryable (reintentar con el mismo operationId). Un 401 invalida el lote completo. Requiere casefile.write. API-041.
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
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", ve.Msg)
		}
		log.Printf("note: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
	}
	return c.JSON(res)
}
