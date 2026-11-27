// Package auditlog expone GET /audit-logs (API-035) y GET /audit-alerts (API-036).
package auditlog

import (
	"errors"
	"log"

	hu "backend-hackathon/internal/handlers/handlerutil"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	almodels "backend-hackathon/internal/models/auditlog"
	repository "backend-hackathon/internal/repository/repositories"
	svc "backend-hackathon/internal/services/auditlog"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ almodels.EntryPage

func mapError(err error) error {
	var he *httpx.Error
	if errors.As(err, &he) {
		return he
	}
	log.Printf("auditlog: %v", err)
	return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
}

func invalid(msg string) error {
	return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", msg)
}

// Logs consulta la bitácora de actividad.
//
// @Summary      Consultar bitácora
// @Description  Eventos de auditoría, del más reciente al más antiguo. La propia consulta queda auditada. Requiere el permiso audit.read. API-035.
// @Tags         audit
// @Produce      json
// @Security     BearerAuth
// @Param        actorId       query  string  false  "UUID del actor"
// @Param        recordId      query  string  false  "UUID del recurso afectado (p. ej. expediente)"
// @Param        action        query  string  false  "Acción exacta, p. ej. casefile.disclose.curp"
// @Param        resourceType  query  string  false  "Tipo de recurso, p. ej. expediente"
// @Param        result        query  string  false  "exito | denegado | error | intento_entrega"
// @Param        from          query  string  false  "Desde (RFC 3339, inclusive)"
// @Param        to            query  string  false  "Hasta (RFC 3339, exclusivo)"
// @Param        limit         query  int     false  "1..100 (20 por defecto)"
// @Param        cursor        query  string  false  "Cursor opaco de la página anterior"
// @Success      200  {object}  almodels.EntryPage
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /audit-logs [get]
func (h *Handler) Logs(c *fiber.Ctx) error {
	actor, err := hu.Actor(c)
	if err != nil {
		return err
	}
	var f repository.AuditoriaFilter
	if f.ActorID, err = hu.UUIDQuery(c, "actorId"); err != nil {
		return err
	}
	if f.RecursoID, err = hu.UUIDQuery(c, "recordId"); err != nil {
		return err
	}
	f.Accion = c.Query("action")
	f.TipoRecurso = c.Query("resourceType")
	if raw := c.Query("result"); raw != "" {
		res := models.ResultadoAuditoria(raw)
		switch res {
		case models.ResultadoExito, models.ResultadoDenegado, models.ResultadoError, models.ResultadoIntentoEntrega:
			f.Resultado = &res
		default:
			return invalid("result inválido")
		}
	}
	if f.Rango, err = hu.Range(c); err != nil {
		return err
	}
	page, err := hu.Page(c)
	if err != nil {
		return err
	}
	res, err := h.service.Logs(c.Context(), actor, f, page)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}

// Alerts consulta las alertas de acceso inusual.
//
// @Summary      Consultar alertas de acceso
// @Description  Alertas generadas por volumen inusual de accesos. La consulta queda auditada. Requiere el permiso audit.read. API-036.
// @Tags         audit
// @Produce      json
// @Security     BearerAuth
// @Param        status   query  string  false  "abierta | revisada"
// @Param        level    query  string  false  "media | alta"
// @Param        actorId  query  string  false  "UUID del actor observado"
// @Param        from     query  string  false  "Desde (RFC 3339)"
// @Param        to       query  string  false  "Hasta (RFC 3339)"
// @Param        limit    query  int     false  "1..100 (20 por defecto)"
// @Param        cursor   query  string  false  "Cursor opaco"
// @Success      200  {object}  almodels.AlertPage
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /audit-alerts [get]
func (h *Handler) Alerts(c *fiber.Ctx) error {
	actor, err := hu.Actor(c)
	if err != nil {
		return err
	}
	var f repository.AlertaFilter
	if f.ActorObservadoID, err = hu.UUIDQuery(c, "actorId"); err != nil {
		return err
	}
	if raw := c.Query("status"); raw != "" {
		st := models.EstadoAlerta(raw)
		if st != models.AlertaAbierta && st != models.AlertaRevisada {
			return invalid("status inválido")
		}
		f.Estado = &st
	}
	if raw := c.Query("level"); raw != "" {
		lv := models.NivelAlerta(raw)
		if lv != models.AlertaMedia && lv != models.AlertaAlta {
			return invalid("level inválido")
		}
		f.Nivel = &lv
	}
	if f.Rango, err = hu.Range(c); err != nil {
		return err
	}
	page, err := hu.Page(c)
	if err != nil {
		return err
	}
	res, err := h.service.Alerts(c.Context(), actor, f, page)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}
