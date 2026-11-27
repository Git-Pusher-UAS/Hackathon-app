package handlers

import (
	"errors"
	"log"

	"backend-hackathon/internal/auth"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/gofiber/fiber/v2"
)

// Los errores salen con el sobre del contrato (httpx.Write en el ErrorHandler).
var _ httpx.ErrorResponse

// AuthHandler expone login, logout, CSRF y MFA.
type AuthHandler struct {
	authService *auth.Service
}

// NewAuthHandler construye el handler de autenticación.
func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// LoginRequest son las credenciales de acceso.
type LoginRequest struct {
	Usuario  string `json:"usuario" example:"admin"`
	Password string `json:"password" example:"admin123"`
}

// Login autentica usuario/contraseña y devuelve un JWT con sesión revocable.
// Si la cuenta tiene MFA activo responde 403 con challenge_token.
//
// @Summary      Iniciar sesión
// @Description  Valida usuario y contraseña, aplica rate limit y crea una sesión revocable. Si la cuenta tiene MFA activo devuelve un challenge_token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "Credenciales de acceso"
// @Success      200      {object}  auth.LoginResult
// @Failure      400      {object}  httpx.ErrorResponse
// @Failure      401      {object}  httpx.ErrorResponse
// @Failure      403      {object}  auth.MFAChallengeResponse
// @Failure      429      {object}  httpx.ErrorResponse
// @Failure      500      {object}  httpx.ErrorResponse
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
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
		return authError(c, err)
	}

	return c.JSON(res)
}

// Me devuelve la cuenta autenticada y sus permisos actuales.
//
// @Summary      Cuenta autenticada
// @Description  Devuelve la cuenta del portador del token y sus permisos efectivos.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  auth.UsuarioInfo
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /auth/me [get]
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
//
// @Summary      Cerrar sesión
// @Description  Revoca la sesión asociada al token actual. Es idempotente.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Param        X-CSRF-Token  header  string  true  "Token CSRF de la sesión (viene en csrf_token del login)"
// @Success      204  "Sin contenido"
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /auth/logout [post]
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

// CSRFToken rota y devuelve el token CSRF de la sesión actual.
//
// @Summary      Obtener token CSRF
// @Description  Genera un token CSRF nuevo para la sesión autenticada e invalida el anterior.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  auth.CSRFResult
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /auth/csrf-token [get]
func (h *AuthHandler) CSRFToken(c *fiber.Ctx) error {
	sessionID, ok := middleware.SessionID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, auth.ErrTokenInvalido.Error())
	}

	res, err := h.authService.RotarCSRF(c.Context(), sessionID)
	if err != nil {
		log.Printf("auth: csrf: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "error interno")
	}

	return c.JSON(res)
}

// EnrollMFA inicia el enrolamiento TOTP.
//
// @Summary      Iniciar enrolamiento MFA
// @Description  Genera el secreto TOTP y la URL otpauth para escanear con una app autenticadora.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Param        X-CSRF-Token  header  string  true  "Token CSRF de la sesión (viene en csrf_token del login)"
// @Success      201  {object}  auth.EnrollmentResult
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      409  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /auth/mfa/enrollments [post]
func (h *AuthHandler) EnrollMFA(c *fiber.Ctx) error {
	personalID, ok := middleware.PersonalID(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, auth.ErrTokenInvalido.Error())
	}

	res, err := h.authService.EnrollMFA(c.Context(), personalID, middleware.RequestID(c), c.IP())
	if err != nil {
		return mfaError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(res)
}

// VerifyMFARequest es el cuerpo de /auth/mfa/verify.
type VerifyMFARequest struct {
	Code           string `json:"code" example:"123456"`
	ChallengeToken string `json:"challenge_token,omitempty"`
}

// VerifyMFA valida un código TOTP.
// Con Bearer confirma el enrolamiento; con challenge_token completa el login.
//
// @Summary      Verificar código MFA
// @Description  Con Bearer confirma el enrolamiento; con challenge_token completa el login y devuelve la sesión.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      VerifyMFARequest  true  "Código y challenge opcional"
// @Success      200      {object}  auth.VerifyMFAResult
// @Failure      400      {object}  httpx.ErrorResponse
// @Failure      401      {object}  httpx.ErrorResponse
// @Failure      403      {object}  httpx.ErrorResponse
// @Failure      429      {object}  httpx.ErrorResponse
// @Failure      500      {object}  httpx.ErrorResponse
// @Router       /auth/mfa/verify [post]
func (h *AuthHandler) VerifyMFA(c *fiber.Ctx) error {
	var req VerifyMFARequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "cuerpo de la petición inválido")
	}

	in := auth.VerifyMFAInput{
		Challenge: req.ChallengeToken,
		Code:      req.Code,
		IP:        c.IP(),
		UserAgent: c.Get(fiber.HeaderUserAgent),
		RequestID: middleware.RequestID(c),
	}
	if id, ok := middleware.PersonalID(c); ok {
		in.PersonalID = &id
	}

	res, err := h.authService.VerifyMFA(c.Context(), in)
	if err != nil {
		return mfaError(err)
	}

	return c.JSON(res)
}

func authError(c *fiber.Ctx, err error) error {
	var ch *auth.MFAChallengeError
	if errors.As(err, &ch) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":           auth.ErrMFARequerido.Error(),
			"challenge_token": ch.Challenge,
		})
	}

	switch {
	case errors.Is(err, auth.ErrCredenciales):
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	case errors.Is(err, auth.ErrSuspendido):
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	case errors.Is(err, auth.ErrBloqueado), errors.Is(err, auth.ErrDemasiadosIntentos):
		return fiber.NewError(fiber.StatusTooManyRequests, err.Error())
	default:
		log.Printf("auth: login: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "error interno")
	}
}

func mfaError(err error) error {
	switch {
	case errors.Is(err, auth.ErrCodigoMFAInvalido),
		errors.Is(err, auth.ErrDesafioInvalido),
		errors.Is(err, auth.ErrTokenInvalido),
		errors.Is(err, repository.ErrNotFound):
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	case errors.Is(err, auth.ErrSuspendido):
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	case errors.Is(err, auth.ErrMFAYaActivo):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, auth.ErrMFANoIniciado):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, auth.ErrDemasiadosIntentos):
		return fiber.NewError(fiber.StatusTooManyRequests, err.Error())
	default:
		log.Printf("auth: mfa: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "error interno")
	}
}
