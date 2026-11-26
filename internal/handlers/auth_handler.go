package handlers

import (
	"errors"
	"log"

	"backend-hackathon/internal/auth"
	"backend-hackathon/internal/middleware"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/gofiber/fiber/v2"
)

// AuthHandler expone login, logout y los datos de la cuenta autenticada.
type AuthHandler struct {
	authService *auth.Service
}

// NewAuthHandler construye el handler de autenticación.
func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type loginRequest struct {
	Usuario  string `json:"usuario"`
	Password string `json:"password"`
}

// Login autentica usuario/contraseña y devuelve un JWT con sesión revocable.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "cuerpo de la petición inválido")
	}

	res, err := h.authService.Login(c.Context(), auth.LoginInput{
		Usuario:   req.Usuario,
		Password:  req.Password,
		IP:        c.IP(),
		UserAgent: c.Get(fiber.HeaderUserAgent),
		RequestID: middleware.RequestID(c),
	})
	if err != nil {
		return authError(err)
	}

	return c.JSON(res)
}

// Me devuelve la cuenta autenticada y sus permisos actuales.
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	personalID, ok := middleware.PersonalID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, auth.ErrTokenInvalido.Error())
	}

	info, err := h.authService.Me(c.Context(), personalID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return fiber.NewError(fiber.StatusUnauthorized, auth.ErrTokenInvalido.Error())
		}
		log.Printf("auth: me: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "error interno")
	}

	return c.JSON(info)
}

// Logout revoca la sesión actual.
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	personalID, ok := middleware.PersonalID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, auth.ErrTokenInvalido.Error())
	}
	sessionID, ok := middleware.SessionID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, auth.ErrTokenInvalido.Error())
	}

	err := h.authService.Logout(
		c.Context(),
		personalID,
		sessionID,
		middleware.RequestID(c),
		c.IP(),
	)
	if err != nil {
		log.Printf("auth: logout: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "error interno")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func authError(err error) error {
	switch {
	case errors.Is(err, auth.ErrCredenciales):
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	case errors.Is(err, auth.ErrSuspendido):
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	case errors.Is(err, auth.ErrMFARequerido):
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	case errors.Is(err, auth.ErrBloqueado), errors.Is(err, auth.ErrDemasiadosIntentos):
		return fiber.NewError(fiber.StatusTooManyRequests, err.Error())
	default:
		log.Printf("auth: login: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "error interno")
	}
}
