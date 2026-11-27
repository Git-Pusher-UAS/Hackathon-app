// Package referral expone la bandeja de canalizaciones /api/v1/referrals.
package referral

import (
	"errors"
	"log"
	"strings"

	hu "backend-hackathon/internal/handlers/handlerutil"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	refmodels "backend-hackathon/internal/models/referral"
	svc "backend-hackathon/internal/services/referral"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ refmodels.Page

// MapError traduce los errores del módulo al catálogo §18.
func MapError(err error) error {
	var he *httpx.Error
	switch {
	case errors.As(err, &he):
		return he
	case errors.Is(err, svc.ErrNotFound):
		return httpx.E(fiber.StatusNotFound, "not_found", "recurso no disponible")
	case errors.Is(err, svc.ErrInvalidTransition):
		return httpx.E(fiber.StatusConflict, "invalid_transition", "transición de estado inválida")
	case errors.Is(err, svc.ErrCaseClosed):
		return httpx.E(fiber.StatusConflict, "record_closed", "el expediente está cerrado")
	case errors.Is(err, svc.ErrPreconditionRequired):
		return httpx.E(fiber.StatusPreconditionRequired, "precondition_required", "If-Match requerido")
	case errors.Is(err, svc.ErrVersionMismatch):
		return httpx.E(fiber.StatusPreconditionFailed, "precondition_failed", "versión obsoleta")
	case errors.Is(err, svc.ErrCryptoUnavailable):
		return httpx.E(fiber.StatusServiceUnavailable, "encryption_unavailable", "cifrado no configurado")
	default:
		log.Printf("referral: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "internal_error", "error interno")
	}
}

// List devuelve la bandeja de canalizaciones del departamento del actor.
//
// @Summary      Bandeja de canalizaciones
// @Description  incoming: canalizaciones a las áreas del actor; outgoing: las que emitieron sus áreas. Con recordId se usa en la pestaña Canalizaciones del expediente. Cada elemento trae versionTag para If-Match y el motivo del último rechazo o cancelación. Requiere referral.read.
// @Tags         referrals
// @Produce      json
// @Security     BearerAuth
// @Param        direction  query  string  false  "incoming (por defecto) | outgoing"
// @Param        status     query  string  false  "Estados separados por coma: pendiente,aceptada,rechazada,completada,cancelada"
// @Param        recordId   query  string  false  "UUID del expediente"
// @Param        limit      query  int     false  "1..100 (20 por defecto)"
// @Param        cursor     query  string  false  "Cursor opaco"
// @Success      200  {object}  refmodels.Page
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      503  {object}  httpx.ErrorResponse
// @Router       /referrals [get]
func (h *Handler) List(c *fiber.Ctx) error {
	actor, err := hu.Actor(c)
	if err != nil {
		return err
	}
	q := refmodels.ListQuery{Direction: c.Query("direction")}
	if raw := c.Query("status"); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			st := models.EstadoCanalizacion(strings.TrimSpace(part))
			switch st {
			case models.CanalizacionPendiente, models.CanalizacionAceptada, models.CanalizacionRechazada,
				models.CanalizacionCancelada, models.CanalizacionCompletada:
				q.Estados = append(q.Estados, st)
			default:
				return httpx.Invalid("status inválido", map[string]string{"status": "valor_invalido"})
			}
		}
	}
	if q.RecordID, err = hu.UUIDQuery(c, "recordId"); err != nil {
		return err
	}
	page, err := hu.Page(c)
	if err != nil {
		return err
	}
	res, err := h.service.List(c.Context(), actor, q, page)
	if err != nil {
		return MapError(err)
	}
	return c.JSON(res)
}

// Transition cambia el estado de una canalización.
//
// @Summary      Cambiar estado de una canalización
// @Description  aceptar (destino, pendiente a aceptada; crea la asignación del área destino), rechazar (destino, motivo obligatorio), completar (destino, aceptada a completada; en una externa la completa el origen desde pendiente), cancelar (origen, pendiente a cancelada, motivo obligatorio). Exige If-Match con el versionTag. Requiere referral.transition.
// @Tags         referrals
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        referralId    path    string                       true  "UUID de la canalización"
// @Param        X-CSRF-Token  header  string                       true  "Token CSRF de la sesión"
// @Param        If-Match      header  string                       true  "versionTag vigente"
// @Param        request       body    refmodels.TransitionRequest  true  "Acción y motivo"
// @Success      200  {object}  refmodels.View
// @Header       200  {string}  ETag  "Nueva versión"
// @Failure      404  {object}  httpx.ErrorResponse
// @Failure      409  {object}  httpx.ErrorResponse  "invalid_transition o record_closed"
// @Failure      412  {object}  httpx.ErrorResponse  "precondition_failed"
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      428  {object}  httpx.ErrorResponse  "precondition_required"
// @Router       /referrals/{referralId}/transitions [post]
func (h *Handler) Transition(c *fiber.Ctx) error {
	actor, err := hu.Actor(c)
	if err != nil {
		return err
	}
	id, err := hu.UUIDParam(c, "referralId")
	if err != nil {
		return err
	}
	var req refmodels.TransitionRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.Transition(c.Context(), actor, id, c.Get(fiber.HeaderIfMatch), req)
	if err != nil {
		return MapError(err)
	}
	c.Set(fiber.HeaderETag, httpx.FormatETag(res.VersionTag))
	return c.JSON(res)
}
