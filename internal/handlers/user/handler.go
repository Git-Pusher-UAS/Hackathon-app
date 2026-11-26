// Package user expone los handlers HTTP de /api/v1/users.
package user

import (
	"errors"
	"log"
	"strconv"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	usermodels "backend-hackathon/internal/models/user"
	repository "backend-hackathon/internal/repository/repositories"
	userservice "backend-hackathon/internal/services/user"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handler agrupa los endpoints de cuentas y sesiones.
type Handler struct {
	service userservice.Service
}

// New construye el handler de usuarios.
func New(service userservice.Service) *Handler {
	return &Handler{service: service}
}

// Me responde API-006.
func (h *Handler) Me(c *fiber.Ctx) error {
	actorID, ok := middleware.PersonalID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesión inválida")
	}
	sessionID, ok := middleware.SessionID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesión inválida")
	}

	result, err := h.service.Me(c.Context(), actorID, sessionID)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(result)
}

// ListMySessions responde API-008.
func (h *Handler) ListMySessions(c *fiber.Ctx) error {
	actorID, ok := middleware.PersonalID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesión inválida")
	}
	sessionID, ok := middleware.SessionID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesión inválida")
	}

	page, err := parsePage(c)
	if err != nil {
		return err
	}
	result, err := h.service.ListMySessions(c.Context(), actorID, sessionID, page)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(result)
}

// RevokeMySession responde API-009.
func (h *Handler) RevokeMySession(c *fiber.Ctx) error {
	actorID, ok := middleware.PersonalID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesión inválida")
	}
	sessionID, err := parseUUIDParam(c, "sessionId")
	if err != nil {
		return err
	}
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesión inválida")
	}

	if err := h.service.RevokeMySession(c.Context(), actor, actorID, sessionID); err != nil {
		return mapError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// List responde API-010.
func (h *Handler) List(c *fiber.Ctx) error {
	actorID, ok := middleware.PersonalID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesión inválida")
	}
	page, err := parsePage(c)
	if err != nil {
		return err
	}

	result, err := h.service.List(c.Context(), actorID, c.Query("q"), page)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(result)
}

// Create responde API-011.
func (h *Handler) Create(c *fiber.Ctx) error {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesión inválida")
	}

	var req usermodels.CreateRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return mapError(err)
	}

	result, err := h.service.Create(c.Context(), actor, req)
	if err != nil {
		return mapError(err)
	}
	c.Set(fiber.HeaderETag, httpx.FormatETag(result.VersionTag))
	return c.Status(fiber.StatusCreated).JSON(result)
}

// Update responde API-012.
func (h *Handler) Update(c *fiber.Ctx) error {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesión inválida")
	}
	targetID, err := parseUUIDParam(c, "userId")
	if err != nil {
		return err
	}

	var req usermodels.UpdateRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return mapError(err)
	}

	ifMatch, ok := httpx.ParseIfMatch(c.Get(fiber.HeaderIfMatch))
	if !ok {
		return httpx.E(fiber.StatusPreconditionRequired, "PRECONDITION_REQUIRED", "If-Match requerido")
	}

	result, err := h.service.Update(c.Context(), actor, targetID, req, ifMatch)
	if err != nil {
		return mapError(err)
	}
	c.Set(fiber.HeaderETag, httpx.FormatETag(result.VersionTag))
	return c.JSON(result)
}

// RevokeAllSessions responde API-013.
func (h *Handler) RevokeAllSessions(c *fiber.Ctx) error {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesión inválida")
	}
	targetID, err := parseUUIDParam(c, "userId")
	if err != nil {
		return err
	}

	var req usermodels.RevokeSessionsRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return mapError(err)
	}

	if err := h.service.RevokeAllSessions(c.Context(), actor, targetID, req.Motivo); err != nil {
		return mapError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ResetMFA responde API-014.
func (h *Handler) ResetMFA(c *fiber.Ctx) error {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesión inválida")
	}
	targetID, err := parseUUIDParam(c, "userId")
	if err != nil {
		return err
	}

	var req usermodels.MFAResetRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return mapError(err)
	}

	resetID, err := h.service.ResetMFA(c.Context(), actor, targetID, req.Motivo, req.VerificacionPresencial)
	if err != nil {
		return mapError(err)
	}
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"resetId": resetID})
}

func parsePage(c *fiber.Ctx) (repository.PageRequest, error) {
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			return repository.PageRequest{}, httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "limit debe estar entre 1 y 100")
		}
		limit = parsed
	}
	return repository.PageRequest{Limit: limit, Cursor: c.Query("cursor")}, nil
}

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(name))
	if err != nil {
		return uuid.Nil, httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", name+" inválido")
	}
	return id, nil
}

func mapError(err error) error {
	var httpErr *httpx.Error
	if errors.As(err, &httpErr) {
		return httpErr
	}

	var validation *userservice.ValidationError
	if errors.As(err, &validation) {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", validation.Msg)
	}

	switch {
	case errors.Is(err, userservice.ErrNotFound):
		return httpx.E(fiber.StatusNotFound, "RESOURCE_NOT_FOUND", "recurso no disponible")
	case errors.Is(err, userservice.ErrDuplicateUsername):
		return httpx.E(fiber.StatusConflict, "DUPLICATE_USERNAME", "el nombre de usuario ya está registrado")
	case errors.Is(err, userservice.ErrLastAdmin):
		return httpx.E(fiber.StatusConflict, "LAST_ACTIVE_ADMIN", "no se puede dejar el sistema sin administrador activo")
	case errors.Is(err, userservice.ErrSelfElevation):
		return httpx.E(fiber.StatusForbidden, "SELF_ELEVATION_FORBIDDEN", "no puede modificar sus propios permisos")
	case errors.Is(err, userservice.ErrVersionMismatch):
		return httpx.E(fiber.StatusPreconditionFailed, "PRECONDITION_FAILED", "versión obsoleta")
	case errors.Is(err, userservice.ErrPreconditionRequired):
		return httpx.E(fiber.StatusPreconditionRequired, "PRECONDITION_REQUIRED", "If-Match requerido")
	case errors.Is(err, userservice.ErrMFAVerificationRequired):
		return httpx.E(fiber.StatusForbidden, "MFA_VERIFICATION_REQUIRED", "verificación presencial requerida")
	case errors.Is(err, userservice.ErrUnauthenticated):
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesión inválida")
	case errors.Is(err, userservice.ErrCryptoUnavailable):
		return httpx.E(fiber.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "cifrado no configurado")
	default:
		log.Printf("usuario: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
	}
}
