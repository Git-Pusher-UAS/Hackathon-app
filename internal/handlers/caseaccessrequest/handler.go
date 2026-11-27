// Package caseaccessrequest expone POST y GET /case-access-requests.
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

func mapError(err error) error {
	var he *httpx.Error
	if errors.As(err, &he) {
		return he
	}
	var ve *svc.ValidationError
	switch {
	case errors.As(err, &ve):
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", ve.Msg)
	case errors.Is(err, svc.ErrNotFound):
		return httpx.E(fiber.StatusNotFound, "RESOURCE_NOT_FOUND", "enlace no disponible")
	case errors.Is(err, svc.ErrGone):
		return httpx.E(fiber.StatusGone, "LINK_EXPIRED", "el enlace venció o ya se usó")
	case errors.Is(err, svc.ErrCryptoUnavailable):
		return httpx.E(fiber.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "cifrado no configurado")
	default:
		log.Printf("caseaccessrequest: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
	}
}

// Create pide acceso a un expediente existente.
//
// @Summary      Solicitar acceso a un expediente
// @Description  Usa el linkToken de un solo uso que devuelve POST /case-files/duplicate-checks cuando la persona ya tiene expediente. El token vence a los 15 minutos y solo sirve a quien lo obtuvo. Requiere casefile.write. API-018.
// @Tags         case-access-requests
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-CSRF-Token  header  string                   true  "Token CSRF de la sesión"
// @Param        request       body    carmodels.CreateRequest  true  "Token y motivo"
// @Success      201  {object}  carmodels.Created
// @Failure      400  {object}  httpx.ErrorResponse
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      404  {object}  httpx.ErrorResponse
// @Failure      410  {object}  httpx.ErrorResponse  "El enlace venció o ya se usó"
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      503  {object}  httpx.ErrorResponse
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
		return mapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(res)
}

// List devuelve la bandeja de solicitudes de acceso.
//
// @Summary      Bandeja de solicitudes de acceso
// @Description  Solicitudes de acceso a expedientes, de la más antigua a la más reciente. Solo metadatos y el motivo; nunca datos de identidad ni notas. Requiere casefile.assign. API-019.
// @Tags         case-access-requests
// @Produce      json
// @Security     BearerAuth
// @Param        status  query  string  false  "pendiente | aprobada | rechazada"
// @Param        limit   query  int     false  "1..100 (20 por defecto)"
// @Param        cursor  query  string  false  "Cursor opaco"
// @Success      200  {object}  carmodels.Page
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      503  {object}  httpx.ErrorResponse
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
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "status inválido")
		}
	}
	page, err := hu.Page(c)
	if err != nil {
		return err
	}
	res, err := h.service.List(c.Context(), actor, estado, page)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}
