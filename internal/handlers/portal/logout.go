package portal

import (
	"strings"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// Logout cierra sesión de portal.
//
// @Summary      Logout portal
// @Description  Revoca la sesión portal y limpia cookies.
// @Tags         portal
// @Produce      json
// @Param        X-CSRF-Token  header  string  true  "Token CSRF portal"
// @Success      204  "Sin contenido"
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /portal/auth/logout [post]
func (h *Handler) Logout(c *fiber.Ctx) error {
	sid, ok := middleware.PortalSessionID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	if err := h.service.Logout(c.Context(), sid); err != nil {
		return mapError(err)
	}
	secure := c.Secure() || strings.EqualFold(c.Get("X-Forwarded-Proto"), "https")
	c.Cookie(&fiber.Cookie{Name: middleware.PortalCookieName, Value: "", Path: "/", HTTPOnly: true, Secure: secure, SameSite: "Lax", MaxAge: -1})
	c.Cookie(&fiber.Cookie{Name: "portal_csrf_token", Value: "", Path: "/", HTTPOnly: false, Secure: secure, SameSite: "Lax", MaxAge: -1})
	return c.SendStatus(fiber.StatusNoContent)
}
