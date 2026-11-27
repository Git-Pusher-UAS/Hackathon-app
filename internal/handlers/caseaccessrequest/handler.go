// Package caseaccessrequest expone /api/case-access-requests.
package caseaccessrequest

import (
	"errors"
	"log"
	"strconv"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	"backend-hackathon/internal/models"
	carmodels "backend-hackathon/internal/models/caseaccessrequest"
	repository "backend-hackathon/internal/repository/repositories"
	svc "backend-hackathon/internal/services/caseaccessrequest"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ carmodels.Item

func actorOf(c *fiber.Ctx) (repository.ActorContext, error) {
	a, ok := middleware.ActorFrom(c)
	if !ok {
		return a, httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	return a, nil
}

func MapError(err error) error {
	var he *httpx.Error
	switch {
	case errors.As(err, &he):
		return he
	case errors.Is(err, svc.ErrLinkInvalid), errors.Is(err, svc.ErrLinkExpired):
		return httpx.E(fiber.StatusGone, "GONE", "el linkToken expiro o es invalido")
	case errors.Is(err, svc.ErrRequestExists):
		return httpx.E(fiber.StatusConflict, "REQUEST_EXISTS", "ya hay una solicitud pendiente para ese expediente")
	case errors.Is(err, svc.ErrNotFound):
		return httpx.E(fiber.StatusNotFound, "RESOURCE_NOT_FOUND", "recurso no disponible")
	case errors.Is(err, svc.ErrNotPending):
		return httpx.E(fiber.StatusConflict, "REQUEST_NOT_PENDING", "la solicitud ya fue resuelta")
	case errors.Is(err, svc.ErrCryptoUnavailable):
		return httpx.E(fiber.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "cifrado no configurado")
	default:
		log.Printf("caseaccessrequest: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
	}
}

// Create solicita acceso a un expediente.
//
// @Summary      Solicitar acceso a expediente
// @Description  Crea una solicitud de acceso usando linkToken.
// @Tags         case-access-requests
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-CSRF-Token  header  string                   true  "Token CSRF"
// @Param        request       body    carmodels.CreateRequest  true  "Solicitud"
// @Success      201  {object}  carmodels.Created
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      409  {object}  httpx.ErrorResponse
// @Failure      410  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      503  {object}  httpx.ErrorResponse
// @Router       /case-access-requests [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	actor, err := actorOf(c)
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

// List devuelve solicitudes de acceso.
//
// @Summary      Listar solicitudes de acceso
// @Description  Devuelve metadatos de solicitudes. Nunca expone identidad PII ni notas.
// @Tags         case-access-requests
// @Produce      json
// @Security     BearerAuth
// @Param        status  query  string  false  "pendiente | aprobada | rechazada"
// @Param        limit   query  int     false  "1..100"
// @Param        cursor  query  string  false  "Cursor"
// @Success      200  {array}   carmodels.Item
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Router       /case-access-requests [get]
func (h *Handler) List(c *fiber.Ctx) error {
	actor, err := actorOf(c)
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
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "status invalido")
		}
	}
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "limit invalido")
		}
		limit = n
	}
	res, err := h.service.List(c.Context(), actor, estado, repository.PageRequest{Limit: limit, Cursor: c.Query("cursor")})
	if err != nil {
		return MapError(err)
	}
	return c.JSON(res)
}

// Reject rechaza una solicitud pendiente.
//
// @Summary      Rechazar solicitud de acceso
// @Description  Rechaza una solicitud pendiente, audita y responde 204.
// @Tags         case-access-requests
// @Accept       json
// @Security     BearerAuth
// @Param        id            path    string                  true  "ID de solicitud"
// @Param        X-CSRF-Token  header  string                  true  "Token CSRF"
// @Param        request       body    carmodels.RejectRequest true  "Motivo"
// @Success      204  "Rechazada"
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      404  {object}  httpx.ErrorResponse
// @Failure      409  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Router       /case-access-requests/{id}/rejection [post]
func (h *Handler) Reject(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "id invalido")
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
