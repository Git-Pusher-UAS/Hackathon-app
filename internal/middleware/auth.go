package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"strings"
	"time"

	"backend-hackathon/internal/auth"
	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// AuthMiddleware valida el JWT y la sesión revocable asociada.
//
// Comprueba firma/expiración, que la sesión exista y no esté revocada ni vencida,
// que el token coincida con su hash, que la cuenta siga activa y que la versión
// de permisos no haya cambiado. Deja el actor en Locals para los handlers.
func AuthMiddleware(secret string, uow repository.UnitOfWork) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !validateStaffAuth(c, secret, uow) {
			return unauthorized(c)
		}
		return c.Next()
	}
}

func validateStaffAuth(c *fiber.Ctx, secret string, uow repository.UnitOfWork) bool {
	raw, ok := bearerToken(c.Get(fiber.HeaderAuthorization))
	if !ok {
		raw = strings.TrimSpace(c.Cookies("sai_session"))
		ok = raw != ""
	}
	if !ok {
		return false
	}

	claims, err := auth.ParseToken(secret, raw)
	if err != nil {
		return false
	}

	personalID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return false
	}

	ctx := c.Context()
	repos := uow.Repositories()
	now := time.Now()

	sesion, err := repos.Sesiones().GetByID(ctx, claims.SessionID)
	if err != nil || sesion.RevocadoEn != nil || !sesion.VenceAbsoluto.After(now) || !sesion.UltimaActividad.Add(auth.IdleTimeout).After(now) {
		return false
	}
	if sesion.TitularID != personalID {
		return false
	}

	hash := sha256.Sum256([]byte(raw))
	if subtle.ConstantTimeCompare(hash[:], sesion.TokenHash) != 1 {
		return false
	}

	personal, err := repos.Personal().GetByID(ctx, personalID)
	if err != nil || personal.Estado != models.PersonalActivo {
		return false
	}
	if personal.BloqueadoHasta != nil && personal.BloqueadoHasta.After(now) {
		return false
	}
	if personal.VersionPermisos != claims.PermVersion {
		return false
	}

	// Actualiza ultima_actividad sin escribir en cada petición.
	if now.Sub(sesion.UltimaActividad) > time.Minute {
		_ = repos.Sesiones().Touch(ctx, sesion.ID, now)
	}

	c.Locals(localPersonalID, personalID)
	c.Locals(localActorID, claims.ActorID)
	c.Locals(localSessionID, sesion.ID)
	c.Locals(localRol, claims.Rol)
	c.Locals(localPermisos, claims.Permisos)
	c.Locals(localCSRFHash, sesion.CSRFHash)
	c.Locals(localActor, repository.ActorContext{
		ActorID:   claims.ActorID,
		RequestID: RequestID(c),
		IPOrigen:  ipPtr(c.IP()),
	})

	return true
}

func bearerToken(header string) (string, bool) {
	parts := strings.SplitN(strings.TrimSpace(header), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}

func unauthorized(c *fiber.Ctx) error {
	c.Set(fiber.HeaderWWWAuthenticate, "Bearer")
	return fiber.NewError(fiber.StatusUnauthorized, auth.ErrTokenInvalido.Error())
}

func ipPtr(ip string) *string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return nil
	}
	return &ip
}
