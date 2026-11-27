package referral

import (
	"errors"
	"log"
	"strconv"
	"strings"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	"backend-hackathon/internal/models"
	refmodels "backend-hackathon/internal/models/referral"
	repository "backend-hackathon/internal/repository/repositories"
	svc "backend-hackathon/internal/services/referral"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ refmodels.View

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
	case errors.Is(err, svc.ErrNotFound):
		return httpx.E(fiber.StatusNotFound, "RESOURCE_NOT_FOUND", "recurso no disponible")
	case errors.Is(err, svc.ErrInvalidTransition):
		return httpx.E(fiber.StatusConflict, "CONFLICT", "transicion de estado invalida")
	case errors.Is(err, svc.ErrCaseClosed):
		return httpx.E(fiber.StatusConflict, "CASE_CLOSED", "el expediente esta cerrado")
	case errors.Is(err, svc.ErrPreconditionRequired):
		return httpx.E(fiber.StatusPreconditionRequired, "PRECONDITION_REQUIRED", "If-Match requerido")
	case errors.Is(err, svc.ErrVersionMismatch):
		return httpx.E(fiber.StatusPreconditionFailed, "PRECONDITION_FAILED", "version obsoleta")
	case errors.Is(err, svc.ErrCryptoUnavailable):
		return httpx.E(fiber.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "cifrado no configurado")
	default:
		log.Printf("referral: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
	}
}

func (h *Handler) List(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	q := refmodels.ListQuery{Direction: c.Query("direction")}
	if raw := c.Query("status"); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			st := models.EstadoCanalizacion(strings.TrimSpace(part))
			switch st {
			case models.CanalizacionPendiente, models.CanalizacionAceptada, models.CanalizacionRechazada, models.CanalizacionCancelada, models.CanalizacionCompletada:
				q.Estados = append(q.Estados, st)
			default:
				return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "status invalido")
			}
		}
	}
	if raw := strings.TrimSpace(c.Query("recordId")); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			q.RecordID = &id
		} else {
			q.RecordRef = raw
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
	res, err := h.service.List(c.Context(), actor, q, repository.PageRequest{Limit: limit, Cursor: c.Query("cursor")})
	if err != nil {
		return MapError(err)
	}
	return c.JSON(res)
}

func (h *Handler) Transition(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("referralId"))
	if err != nil {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "referralId invalido")
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
