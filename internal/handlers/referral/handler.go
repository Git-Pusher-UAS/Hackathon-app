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
