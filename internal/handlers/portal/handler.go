package portal

import (
	"errors"
	"log"
	"strings"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	pmodels "backend-hackathon/internal/models/portal"
	svc "backend-hackathon/internal/services/portal"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

func setPortalCookies(c *fiber.Ctx, token, csrf string) {
	secure := c.Secure() || strings.EqualFold(c.Get("X-Forwarded-Proto"), "https")
	c.Cookie(&fiber.Cookie{Name: middleware.PortalCookieName, Value: token, Path: "/", HTTPOnly: true, Secure: secure, SameSite: "Lax", MaxAge: int(24 * time.Hour / time.Second)})
	if csrf != "" {
		c.Cookie(&fiber.Cookie{Name: "portal_csrf_token", Value: csrf, Path: "/", HTTPOnly: false, Secure: secure, SameSite: "Lax", MaxAge: int(24 * time.Hour / time.Second)})
	}
}

func mapError(err error) error {
	var he *httpx.Error
	switch {
	case errors.As(err, &he):
		return he
	case errors.Is(err, svc.ErrNotFound):
		return httpx.E(fiber.StatusNotFound, "RESOURCE_NOT_FOUND", "recurso no disponible")
	case errors.Is(err, svc.ErrInvalidCode):
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "codigo invalido")
	case errors.Is(err, svc.ErrChallengeGone):
		return httpx.E(fiber.StatusGone, "GONE", "desafio expirado o invalido")
	case errors.Is(err, svc.ErrConflict):
		return httpx.E(fiber.StatusConflict, "CONFLICT", "la cuenta ya existe")
	default:
		log.Printf("portal: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
	}
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var req pmodels.LoginRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.Login(c.Context(), req.Identificador)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}

func (h *Handler) CreateAccount(c *fiber.Ctx) error {
	var req pmodels.CreateAccountRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.CreateAccount(c.Context(), req)
	if err != nil {
		return mapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(res)
}

func (h *Handler) Verify(c *fiber.Ctx) error {
	var req pmodels.VerifyRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.Verify(c.Context(), req.ChallengeID, req.Code)
	if err != nil {
		return mapError(err)
	}
	setPortalCookies(c, res.Token, res.CSRFToken)
	return c.JSON(res.Response)
}

func (h *Handler) Me(c *fiber.Ctx) error {
	cuentaID, ok := middleware.PortalCuentaID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	res, err := h.service.Me(c.Context(), cuentaID)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}

func (h *Handler) PatchMe(c *fiber.Ctx) error {
	cuentaID, ok := middleware.PortalCuentaID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	var req pmodels.PatchMeRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.PatchMe(c.Context(), cuentaID, req)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}

func (h *Handler) CreateAlert(c *fiber.Ctx) error {
	cuentaID, ok := middleware.PortalCuentaID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	var req pmodels.CreateAlertRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.CreateAlert(c.Context(), cuentaID, req)
	if err != nil {
		return mapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(res)
}

func (h *Handler) NotifyTrustedContacts(c *fiber.Ctx) error {
	cuentaID, ok := middleware.PortalCuentaID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "id invalido")
	}
	res, err := h.service.NotifyTrustedContacts(c.Context(), cuentaID, id)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}

func (h *Handler) CancelAlert(c *fiber.Ctx) error {
	cuentaID, ok := middleware.PortalCuentaID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "id invalido")
	}
	var req pmodels.CancelAlertRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.CancelAlert(c.Context(), cuentaID, id, req.Motivo)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}
