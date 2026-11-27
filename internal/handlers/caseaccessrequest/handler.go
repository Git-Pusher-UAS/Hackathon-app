// Package caseaccessrequest expone POST y GET /api/v1/case-access-requests.
package caseaccessrequest

import (
	"errors"
	"log"

	hu "backend-hackathon/internal/handlers/handlerutil"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	carmodels "backend-hackathon/internal/models/caseaccessrequest"
	svc "backend-hackathon/internal/services/caseaccessrequest"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ carmodels.Page

// MapError traduce los errores del módulo al catálogo §18.
func MapError(err error) error {
	var he *httpx.Error
	switch {
	case errors.As(err, &he):
		return he
	case errors.Is(err, svc.ErrLinkInvalid):
		return httpx.E(fiber.StatusNotFound, "not_found", "enlace no disponible")
	case errors.Is(err, svc.ErrLinkExpired):
		return httpx.E(fiber.StatusGone, "link_expired", "el enlace venció o ya se usó")
	case errors.Is(err, svc.ErrRequestExists):
		return httpx.E(fiber.StatusConflict, "request_exists", "ya hay una solicitud pendiente para ese expediente")
	case errors.Is(err, svc.ErrNotFound):
		return httpx.E(fiber.StatusNotFound, "not_found", "recurso no disponible")
	case errors.Is(err, svc.ErrNotPending):
		return httpx.E(fiber.StatusConflict, "request_not_pending", "la solicitud ya fue resuelta")
	case errors.Is(err, svc.ErrCryptoUnavailable):
		return httpx.E(fiber.StatusServiceUnavailable, "encryption_unavailable", "cifrado no configurado")
	default:
		log.Printf("caseaccessrequest: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "internal_error", "error interno")
	}
}

// Create pide acceso a un expediente que lleva otra área.
//
// @Summary      Solicitar acceso a un expediente
// @Description  Con el linkToken que devolvió POST /case-files/duplicate-checks, sin volver a enviar la CURP. El token es de un solo uso y solo sirve a quien lo pidió. Coordinación decide en la bandeja sin ver el expediente. Requiere caseaccess.request.
// @Tags         case-access-requests
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-CSRF-Token     header  string                   true   "Token CSRF de la sesión"
// @Param        Idempotency-Key  header  string                   false  "UUID de la operación"
// @Param        request          body    carmodels.CreateRequest  true   "Token y motivo (10 a 500 caracteres)"
// @Success      201  {object}  carmodels.Created
// @Failure      404  {object}  httpx.ErrorResponse  "not_found: token inexistente o de otra persona"
// @Failure      409  {object}  httpx.ErrorResponse  "request_exists"
// @Failure      410  {object}  httpx.ErrorResponse  "link_expired"
// @Failure      422  {object}  httpx.ErrorResponse  "validation_failed con fields"
// @Failure      503  {object}  httpx.ErrorResponse  "encryption_unavailable"
// @Router       /case-access-requests [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	actor, err := hu.Actor(c)
	if err != nil {
		return err
	}
	var req carmodels.CreateRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.Create(c.Context(), actor, req)
	if err != nil {
		return MapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(res)
}

// Reject rechaza una solicitud pendiente.
//
// @Summary      Rechazar solicitud de acceso
// @Description  Rechaza una solicitud pendiente. Para aprobarla se asigna con solicitudId en POST /case-files/{recordId}/assignments. Requiere caseaccess.review.
// @Tags         case-access-requests
// @Accept       json
// @Security     BearerAuth
// @Param        id            path    string         true  "UUID de la solicitud"
// @Param        X-CSRF-Token  header  string         true  "Token CSRF de la sesión"
// @Param        request       body    carmodels.RejectRequest  true  "Motivo (10 a 500 caracteres)"
// @Success      204  "Rechazada"
// @Failure      404  {object}  httpx.ErrorResponse
// @Failure      409  {object}  httpx.ErrorResponse  "request_not_pending"
// @Failure      422  {object}  httpx.ErrorResponse
// @Router       /case-access-requests/{id}/rejection [post]
func (h *Handler) Reject(c *fiber.Ctx) error {
	actor, err := hu.Actor(c)
	if err != nil {
		return err
	}
	id, err := hu.UUIDParam(c, "id")
	if err != nil {
		return err
	}
	var req carmodels.RejectRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	if err := h.service.Reject(c.Context(), actor, id, req.Motivo); err != nil {
		return MapError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// List devuelve la bandeja de solicitudes de acceso.
//
// @Summary      Bandeja de solicitudes de acceso
// @Description  Solicitudes de la más antigua a la más reciente. Solo metadatos y el motivo: nunca datos de identidad de la persona ni notas. La consulta queda en la bitácora. Requiere caseaccess.review.
// @Tags         case-access-requests
// @Produce      json
// @Security     BearerAuth
// @Param        status  query  string  false  "pendiente | aprobada | rechazada"
// @Param        limit   query  int     false  "1..100 (20 por defecto)"
// @Param        cursor  query  string  false  "Cursor opaco"
// @Success      200  {object}  carmodels.Page
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Router       /case-access-requests [get]
func (h *Handler) List(c *fiber.Ctx) error {
	actor, err := hu.Actor(c)
	if err != nil {
		return err
	}
	var estado *models.EstadoSolicitudVinculacion
	if raw := c.Query("status"); raw != "" {
		st := models.EstadoSolicitudVinculacion(raw)
		switch st {
		case models.SolicitudPendiente, models.SolicitudAprobada, models.SolicitudRechazada:
			estado = &st
		default:
			return httpx.Invalid("status inválido", map[string]string{"status": "valor_invalido"})
		}
	}
	page, err := hu.Page(c)
	if err != nil {
		return err
	}
	res, err := h.service.List(c.Context(), actor, estado, page)
	if err != nil {
		return MapError(err)
	}
	return c.JSON(res)
}
