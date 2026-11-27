package handlers

import (
	"backend-hackathon/internal/httpx"
	"errors"
	"log"
	"strings"
	"time"

	"backend-hackathon/internal/auth"
	"backend-hackathon/internal/middleware"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/gofiber/fiber/v2"
)

const (
	sessionCookieName = "sai_session"
	csrfCookieName    = "csrf_token"
)

// AuthHandler expone login, logout, CSRF y MFA.
type AuthHandler struct {
	authService *auth.Service
}

// NewAuthHandler construye el handler de autenticación.
func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func setSessionCookies(c *fiber.Ctx, res *auth.LoginResult) {
	if res == nil {
		return
	}
	secure := c.Secure() || strings.EqualFold(c.Get("X-Forwarded-Proto"), "https")
	sameSite := "Lax"
	c.Cookie(&fiber.Cookie{
		Name:     sessionCookieName,
		Value:    res.Token,
		Path:     "/",
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   int(res.ExpiresIn),
	})
	if res.CSRF != "" {
		c.Cookie(&fiber.Cookie{
			Name:     csrfCookieName,
			Value:    res.CSRF,
			Path:     "/",
			HTTPOnly: false,
			Secure:   secure,
			SameSite: sameSite,
			MaxAge:   int(res.ExpiresIn),
		})
	}
}

func clearSessionCookies(c *fiber.Ctx) {
	secure := c.Secure() || strings.EqualFold(c.Get("X-Forwarded-Proto"), "https")
	for _, name := range []string{sessionCookieName, csrfCookieName} {
		c.Cookie(&fiber.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HTTPOnly: name == sessionCookieName,
			Secure:   secure,
			SameSite: "Lax",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
		})
	}
}

// LoginRequest son las credenciales de acceso.
type LoginRequest struct {
	Username string `json:"username" example:"admin"`
	Usuario  string `json:"usuario" example:"admin"`
	Password string `json:"password" example:"admin123"`
}

// Login autentica usuario/contraseña y devuelve un JWT con sesión revocable.
// Si la cuenta requiere MFA responde 202 con challengeId y next.
//
// @Summary      Iniciar sesión
// @Description  Valida usuario y contraseña. Si requiere MFA devuelve 202 con challengeId y next; si no, emite sesión completa.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "Credenciales de acceso"
// @Success      200      {object}  auth.LoginResult
// @Success      202      {object}  auth.MFAChallengeResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      429      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "cuerpo de la petición inválido")
	}

	username := req.Username
	if username == "" {
		username = req.Usuario
	}

	res, err := h.authService.Login(c.Context(), auth.LoginInput{
		Usuario:   username,
		Password:  req.Password,
		IP:        c.IP(),
		UserAgent: c.Get(fiber.HeaderUserAgent),
		RequestID: middleware.RequestID(c),
	})
	if err != nil {
		return authError(c, err)
	}

	setSessionCookies(c, res)
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
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
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
// @Success      204  "Sin contenido"
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	personalID, ok := middleware.PersonalID(c)
	sessionID, sok := middleware.SessionID(c)
	if ok && sok {
		if err := h.authService.Logout(
			c.Context(),
			personalID,
			sessionID,
			middleware.RequestID(c),
			c.IP(),
		); err != nil {
			log.Printf("auth: logout: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "error interno")
		}
	}

	clearSessionCookies(c)
	return c.SendStatus(fiber.StatusNoContent)
}

// CSRFToken rota y devuelve el token CSRF de la sesión actual o anónima.
//
// @Summary      Obtener token CSRF
// @Description  Devuelve un token CSRF. Si hay sesión autenticada, lo rota en servidor; si es anónimo, emite un token para uso previo al login.
// @Tags         auth
// @Produce      json
// @Success      200  {object}  auth.CSRFResult
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/csrf-token [get]
func (h *AuthHandler) CSRFToken(c *fiber.Ctx) error {
	if sessionID, ok := middleware.SessionID(c); ok {
		res, err := h.authService.RotarCSRF(c.Context(), sessionID)
		if err != nil {
			log.Printf("auth: csrf: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "error interno")
		}
		secure := c.Secure() || strings.EqualFold(c.Get("X-Forwarded-Proto"), "https")
		c.Cookie(&fiber.Cookie{
			Name:     csrfCookieName,
			Value:    res.CSRF,
			Path:     "/",
			HTTPOnly: false,
			Secure:   secure,
			SameSite: "Lax",
		})
		return c.JSON(res)
	}

	token, err := auth.RandomToken(32)
	if err != nil {
		log.Printf("auth: csrf anon: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "error interno")
	}
	secure := c.Secure() || strings.EqualFold(c.Get("X-Forwarded-Proto"), "https")
	c.Cookie(&fiber.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HTTPOnly: false,
		Secure:   secure,
		SameSite: "Lax",
	})
	return c.JSON(auth.CSRFResult{CSRF: token})
}

// EnrollMFA inicia el enrolamiento TOTP.
//
// @Summary      Iniciar enrolamiento MFA
// @Description  Genera el secreto TOTP y la URL otpauth para escanear con una app autenticadora.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      201  {object}  auth.EnrollmentResult
// @Failure      401  {object}  ErrorResponse
// @Failure      409  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
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
	ChallengeID    string `json:"challengeId,omitempty"`
	ChallengeToken string `json:"challenge_token,omitempty"`
}

// VerifyMFA valida un código TOTP.
// Con Bearer confirma el enrolamiento; con challengeId completa el login.
//
// @Summary      Verificar código MFA
// @Description  Con Bearer confirma el enrolamiento; con challengeId completa el login y emite cookies de sesión.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      VerifyMFARequest  true  "Código y challenge opcional"
// @Success      200      {object}  auth.VerifyMFAResult
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Failure      429      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /auth/mfa/verify [post]
func (h *AuthHandler) VerifyMFA(c *fiber.Ctx) error {
	var req VerifyMFARequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "cuerpo de la petición inválido")
	}

	challenge := req.ChallengeID
	if challenge == "" {
		challenge = req.ChallengeToken
	}

	in := auth.VerifyMFAInput{
		Challenge: challenge,
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

	if res.Login != nil {
		setSessionCookies(c, res.Login)
		return c.JSON(fiber.Map{
			"id":                       res.Login.Usuario.ID,
			"usuario":                  res.Login.Usuario.Usuario,
			"nombre":                   res.Login.Usuario.NombreLaboral,
			"nombre_laboral":           res.Login.Usuario.NombreLaboral,
			"rol":                      res.Login.Usuario.Rol,
			"permisos":                 res.Login.Usuario.Permisos,
			"requiere_cambio_password": res.Login.Usuario.RequiereCambioPassword,
			"csrf_token":               res.Login.CSRF,
			"token":                    res.Login.Token,
			"token_type":               res.Login.TokenType,
			"expires_in":               res.Login.ExpiresIn,
		})
	}

	return c.JSON(res)
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
// @Description  El personal elige su contraseña con el código entregado por TI. El código es de un solo uso y vence en 72 h.
// @Tags         auth
// @Accept       json
// @Param        request  body  ActivationRequest  true  "Usuario, código y contraseña nueva"
// @Success      204  "Contraseña fijada"
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      429  {object}  httpx.ErrorResponse
// @Router       /auth/activations [post]
func (h *AuthHandler) Activate(c *fiber.Ctx) error {
	var req ActivationRequest
	if err := c.BodyParser(&req); err != nil {
		return httpx.E(fiber.StatusBadRequest, "INVALID_JSON", "cuerpo de la petición inválido")
	}
	err := h.authService.Activar(c.Context(), auth.ActivacionInput{
		Usuario: req.Usuario, Codigo: req.Codigo, Password: req.Password,
		IP: c.IP(), RequestID: middleware.RequestID(c),
	})
	switch {
	case err == nil:
		return c.SendStatus(fiber.StatusNoContent)
	case errors.Is(err, auth.ErrActivacionInvalida):
		return httpx.E(fiber.StatusUnauthorized, "INVALID_CODE", "código de activación inválido o vencido")
	case errors.Is(err, auth.ErrPasswordInvalida):
		return httpx.Invalid(err.Error(), map[string]string{"password": "longitud_invalida"})
	case errors.Is(err, auth.ErrDemasiadosIntentos):
		return httpx.E(fiber.StatusTooManyRequests, "TOO_MANY_ATTEMPTS", "demasiados intentos")
	default:
		log.Printf("auth: activation: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
	}
}

func authError(c *fiber.Ctx, err error) error {
	var ch *auth.MFAChallengeError
	if errors.As(err, &ch) {
		next := ch.Next
		if next == "" {
			next = "totp"
		}
		return c.Status(fiber.StatusAccepted).JSON(auth.MFAChallengeResponse{
			ChallengeID:    ch.Challenge,
			Next:           next,
			ChallengeToken: ch.Challenge,
			Error:          auth.ErrMFARequerido.Error(),
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
