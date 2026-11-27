package handlers

import (
	"errors"
	"log"
	"time"

	"backend-hackathon/internal/auth"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// retryAfterSeconds es la ventana de bloqueo de login y MFA.
const retryAfterSeconds = 15 * 60

// AuthHandler expone login, segundo factor, CSRF y logout del personal.
type AuthHandler struct {
	authService *auth.Service
}

// NewAuthHandler construye el handler de autenticación.
func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// LoginRequest son las credenciales de acceso.
type LoginRequest struct {
	Usuario  string `json:"usuario" example:"lramos"`
	Password string `json:"password" example:"••••••••••••"`
}

// EnrollMFARequest inicia la configuración del segundo factor.
type EnrollMFARequest struct {
	ChallengeID string `json:"challengeId"`
}

// VerifyMFARequest es el cuerpo de /auth/mfa/verify.
type VerifyMFARequest struct {
	ChallengeID  string     `json:"challengeId"`
	Code         string     `json:"code" example:"123456"`
	EnrollmentID *uuid.UUID `json:"enrollmentId,omitempty"`
}

// ActivationRequest es el cuerpo de POST /auth/activations.
type ActivationRequest struct {
	Usuario  string `json:"usuario" example:"mperez"`
	Codigo   string `json:"codigo" example:"K7P2-9QX4"`
	Password string `json:"password"`
}

// Activate fija la contraseña de una cuenta nueva con su código de activación.
//
// @Summary      Activar cuenta
// @Description  La persona elige su contraseña con el código que TI le entregó al crear la cuenta (POST /users). El código es de un solo uso y vence a las 72 h. Después entra con POST /auth/login y configura su segundo factor. Pública, sin CSRF. No revela si el usuario existe.
// @Tags         auth
// @Accept       json
// @Param        request  body  ActivationRequest  true  "Usuario, código y contraseña nueva"
// @Success      204  "Contraseña fijada"
// @Failure      401  {object}  httpx.ErrorResponse  "invalid_code"
// @Failure      422  {object}  httpx.ErrorResponse  "validation_failed (contraseña de 12 a 128 caracteres)"
// @Failure      429  {object}  httpx.ErrorResponse  "too_many_attempts"
// @Router       /auth/activations [post]
func (h *AuthHandler) Activate(c *fiber.Ctx) error {
	var req ActivationRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	err := h.authService.Activar(c.Context(), auth.ActivacionInput{Usuario: req.Usuario, Codigo: req.Codigo,
		Password: req.Password, IP: c.IP(), RequestID: middleware.RequestID(c)})
	switch {
	case err == nil:
		return c.SendStatus(fiber.StatusNoContent)
	case errors.Is(err, auth.ErrActivacionInvalida):
		return httpx.E(fiber.StatusUnauthorized, "invalid_code", "código de activación inválido o vencido")
	case errors.Is(err, auth.ErrPasswordInvalida):
		return httpx.Invalid(err.Error(), map[string]string{"password": "longitud_invalida"})
	default:
		return authError(err)
	}
}

// setSessionCookie guarda el JWT en sai_staff: HttpOnly, Secure, SameSite=Strict.
func setSessionCookie(c *fiber.Ctx, token string, expires time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     middleware.StaffCookie,
		Value:    token,
		Path:     "/api",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		Secure:   true,
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteStrictMode,
	})
}

func clearSessionCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     middleware.StaffCookie,
		Value:    "",
		Path:     "/api",
		MaxAge:   -1, // se envía como Max-Age=0
		Expires:  time.Unix(0, 0),
		Secure:   true,
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteStrictMode,
	})
}

// Login valida usuario y contraseña. Nunca inicia sesión: responde un reto.
//
// @Summary      Iniciar sesión (paso 1)
// @Description  Valida usuario y contraseña con límite de 5 fallos en 15 min. Responde siempre 202 con un reto: next=totp si la cuenta ya tiene segundo factor, next=enroll si debe configurarlo. Nadie entra sin segundo factor. Pública, sin CSRF.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "Credenciales"
// @Success      202      {object}  auth.Challenge
// @Failure      400      {object}  httpx.ErrorResponse
// @Failure      401      {object}  httpx.ErrorResponse  "invalid_credentials (no revela si el usuario existe)"
// @Failure      403      {object}  httpx.ErrorResponse  "account_suspended"
// @Failure      429      {object}  httpx.ErrorResponse  "too_many_attempts, con Retry-After"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
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
	return c.Status(fiber.StatusAccepted).JSON(res)
}

// EnrollMFA genera el secreto TOTP para una cuenta con next=enroll.
//
// @Summary      Configurar segundo factor
// @Description  Con el challengeId de un login con next=enroll genera el secreto TOTP. otpauthUri se muestra una sola vez; después se confirma con POST /auth/mfa/verify enviando enrollmentId. Pública, sin CSRF.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      EnrollMFARequest  true  "Reto del login"
// @Success      201      {object}  auth.EnrollmentResult
// @Failure      409      {object}  httpx.ErrorResponse  "mfa_already_enrolled"
// @Failure      410      {object}  httpx.ErrorResponse  "challenge_expired"
// @Router       /auth/mfa/enrollments [post]
func (h *AuthHandler) EnrollMFA(c *fiber.Ctx) error {
	var req EnrollMFARequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.authService.EnrollMFA(c.Context(), req.ChallengeID, middleware.RequestID(c), c.IP())
	if err != nil {
		return authError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(res)
}

// VerifyMFA valida el código TOTP y crea la sesión.
//
// @Summary      Verificar segundo factor (paso 2)
// @Description  Valida el código del reto. Si el reto es de enrolamiento, enrollmentId es obligatorio y activa el segundo factor. Responde la sesión y pone la cookie sai_staff (HttpOnly, Secure, SameSite=Strict, Path=/api). El csrfToken se guarda solo en memoria. Pública, sin CSRF.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      VerifyMFARequest  true  "Reto y código"
// @Success      200      {object}  auth.Session
// @Header       200      {string}  Set-Cookie  "sai_staff=<jwt>; HttpOnly; Secure; SameSite=Strict; Path=/api"
// @Failure      401      {object}  httpx.ErrorResponse  "invalid_code"
// @Failure      410      {object}  httpx.ErrorResponse  "challenge_expired"
// @Failure      429      {object}  httpx.ErrorResponse  "too_many_attempts"
// @Router       /auth/mfa/verify [post]
func (h *AuthHandler) VerifyMFA(c *fiber.Ctx) error {
	var req VerifyMFARequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.authService.VerifyMFA(c.Context(), auth.VerifyMFAInput{
		Challenge:    req.ChallengeID,
		EnrollmentID: req.EnrollmentID,
		Code:         req.Code,
		IP:           c.IP(),
		UserAgent:    c.Get(fiber.HeaderUserAgent),
		RequestID:    middleware.RequestID(c),
	})
	if err != nil {
		return authError(err)
	}
	setSessionCookie(c, res.Token, res.ExpiresAt)
	return c.JSON(res)
}

// Me devuelve la cuenta autenticada y sus permisos actuales.
//
// @Summary      Cuenta autenticada (legado)
// @Description  Misma información que el campo usuario de la sesión. El frontend usa GET /users/me.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  auth.UsuarioInfo
// @Failure      401  {object}  httpx.ErrorResponse
// @Router       /auth/me [get]
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	personalID, ok := middleware.PersonalID(c)
	if !ok {
		return sessionExpired()
	}
	info, err := h.authService.Me(c.Context(), personalID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return sessionExpired()
		}
		return internalError("auth: me", err)
	}
	return c.JSON(info)
}

// Logout revoca la sesión actual y expira la cookie.
//
// @Summary      Cerrar sesión
// @Description  Revoca la sesión actual y expira la cookie sai_staff (Max-Age=0). Es idempotente.
// @Tags         auth
// @Security     BearerAuth
// @Param        X-CSRF-Token  header  string  true  "Token CSRF de la sesión"
// @Success      204  "Sin contenido"
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse  "csrf_invalid"
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	personalID, ok := middleware.PersonalID(c)
	if !ok {
		return sessionExpired()
	}
	sessionID, ok := middleware.SessionID(c)
	if !ok {
		return sessionExpired()
	}
	if err := h.authService.Logout(c.Context(), personalID, sessionID, middleware.RequestID(c), c.IP()); err != nil {
		return internalError("auth: logout", err)
	}
	clearSessionCookie(c)
	return c.SendStatus(fiber.StatusNoContent)
}

// CSRFToken rota y devuelve el token CSRF de la sesión actual.
//
// @Summary      Obtener token CSRF
// @Description  Genera un token CSRF nuevo para la sesión e invalida el anterior. La pantalla lo pide al recargar con una sesión viva.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  auth.CSRFResult
// @Failure      401  {object}  httpx.ErrorResponse
// @Router       /auth/csrf-token [get]
func (h *AuthHandler) CSRFToken(c *fiber.Ctx) error {
	sessionID, ok := middleware.SessionID(c)
	if !ok {
		return sessionExpired()
	}
	res, err := h.authService.RotarCSRF(c.Context(), sessionID)
	if err != nil {
		return internalError("auth: csrf", err)
	}
	return c.JSON(res)
}

func sessionExpired() error {
	return httpx.E(fiber.StatusUnauthorized, "session_expired", "sesión inválida o vencida")
}

func internalError(where string, err error) error {
	log.Printf("%s: %v", where, err)
	return httpx.E(fiber.StatusInternalServerError, "internal_error", "error interno")
}

// authError traduce los errores de auth al catálogo §18.
func authError(err error) error {
	var he *httpx.Error
	switch {
	case errors.As(err, &he):
		return he
	case errors.Is(err, auth.ErrCredenciales):
		return httpx.E(fiber.StatusUnauthorized, "invalid_credentials", "usuario o contraseña incorrectos")
	case errors.Is(err, auth.ErrSuspendido):
		return httpx.E(fiber.StatusForbidden, "account_suspended", "cuenta suspendida")
	case errors.Is(err, auth.ErrBloqueado), errors.Is(err, auth.ErrDemasiadosIntentos):
		e := httpx.E(fiber.StatusTooManyRequests, "too_many_attempts", "demasiados intentos")
		e.RetryAfter = retryAfterSeconds
		return e
	case errors.Is(err, auth.ErrCodigoMFAInvalido):
		return httpx.E(fiber.StatusUnauthorized, "invalid_code", "código incorrecto, vencido o repetido")
	case errors.Is(err, auth.ErrDesafioInvalido), errors.Is(err, auth.ErrMFANoIniciado):
		return httpx.E(fiber.StatusGone, "challenge_expired", "el reto venció; vuelve a iniciar sesión")
	case errors.Is(err, auth.ErrMFAYaActivo):
		return httpx.E(fiber.StatusConflict, "mfa_already_enrolled", "la cuenta ya tiene segundo factor")
	default:
		return internalError("auth", err)
	}
}
