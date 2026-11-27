// Package auditlog expone GET /api/v1/audit-logs.
package auditlog

import (
	"errors"
	"log"

	hu "backend-hackathon/internal/handlers/handlerutil"
	"backend-hackathon/internal/httpx"
	almodels "backend-hackathon/internal/models/auditlog"
	svc "backend-hackathon/internal/services/auditlog"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ almodels.Page

// Logs consulta la bitácora de actividad.
//
// @Summary      Consultar bitácora
// @Description  Eventos del más reciente al más antiguo: actor, rol, acción, expediente, resultado y detalle (por ejemplo, el campo revelado y el motivo). Nunca incluye la IP ni valores sensibles. La consulta queda en la bitácora. Requiere auditlog.read.
// @Tags         audit
// @Produce      json
// @Security     BearerAuth
// @Param        actorId   query  string  false  "UUID del actor"
// @Param        recordId  query  string  false  "UUID del expediente"
// @Param        action    query  string  false  "Acción exacta, p. ej. casefile.disclose.telefono"
// @Param        from      query  string  false  "Desde (RFC 3339)"
// @Param        to        query  string  false  "Hasta (RFC 3339, exclusivo)"
// @Param        limit     query  int     false  "1..100 (20 por defecto)"
// @Param        cursor    query  string  false  "Cursor opaco"
// @Success      200  {object}  almodels.Page
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Router       /audit-logs [get]
func (h *Handler) Logs(c *fiber.Ctx) error {
	actor, err := hu.Actor(c)
	if err != nil {
		return err
	}
	var q svc.Query
	if q.ActorID, err = hu.UUIDQuery(c, "actorId"); err != nil {
		return err
	}
	if q.RecordID, err = hu.UUIDQuery(c, "recordId"); err != nil {
		return err
	}
	q.Action = c.Query("action")
	if q.Rango, err = hu.Range(c); err != nil {
		return err
	}
	page, err := hu.Page(c)
	if err != nil {
		return err
	}
	res, err := h.service.Logs(c.Context(), actor, q, page)
	if err != nil {
		var he *httpx.Error
		if errors.As(err, &he) {
			return he
		}
		log.Printf("auditlog: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "internal_error", "error interno")
	}
	return c.JSON(res)
}
