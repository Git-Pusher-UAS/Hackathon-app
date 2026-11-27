package middleware

import (
	"backend-hackathon/internal/repository/repositories"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/google/uuid"
)

// Claves de Locals donde AuthMiddleware guarda el contexto de la petición.
const (
	localPersonalID = "personalID"
	localActorID    = "actorID"
	localSessionID  = "sessionID"
	localRol        = "rol"
	localPermisos   = "permisos"
	localActor      = "actor"
	localCSRFHash   = "csrfHash"

	localPortalCuentaID    = "portalCuentaID"
	localPortalSessionID   = "portalSessionID"
	localPortalCSRFHash    = "portalCSRFHash"
	localEmpleadorID       = "empleadorID"
	localEmpleadorSesionID = "empleadorSessionID"
	localEmpleadorCSRFHash = "empleadorCSRFHash"
)

// RequestID devuelve el id de correlación de Fiber como UUID.
func RequestID(c *fiber.Ctx) uuid.UUID {
	raw, _ := c.Locals(requestid.ConfigDefault.ContextKey).(string)
	if id, err := uuid.Parse(raw); err == nil {
		return id
	}
	return uuid.New()
}

// PersonalID devuelve el id de identidad.personal autenticado.
func PersonalID(c *fiber.Ctx) (uuid.UUID, bool) {
	id, ok := c.Locals(localPersonalID).(uuid.UUID)
	return id, ok
}

// SessionID devuelve el id de la sesión autenticada.
func SessionID(c *fiber.Ctx) (uuid.UUID, bool) {
	id, ok := c.Locals(localSessionID).(uuid.UUID)
	return id, ok
}

// Rol devuelve el código del rol autenticado.
func Rol(c *fiber.Ctx) (string, bool) {
	rol, ok := c.Locals(localRol).(string)
	return rol, ok
}

// Permisos devuelve los códigos de permiso efectivos del token.
func Permisos(c *fiber.Ctx) []string {
	perms, _ := c.Locals(localPermisos).([]string)
	return perms
}

// ActorFrom devuelve el contexto de actor para abrir una transacción con RLS.
func ActorFrom(c *fiber.Ctx) (repository.ActorContext, bool) {
	actor, ok := c.Locals(localActor).(repository.ActorContext)
	return actor, ok
}

// CSRFHash devuelve el hash del token CSRF ligado a la sesión autenticada.
func CSRFHash(c *fiber.Ctx) ([]byte, bool) {
	hash, ok := c.Locals(localCSRFHash).([]byte)
	return hash, ok
}

func PortalCuentaID(c *fiber.Ctx) (uuid.UUID, bool) {
	id, ok := c.Locals(localPortalCuentaID).(uuid.UUID)
	return id, ok
}

func PortalSessionID(c *fiber.Ctx) (uuid.UUID, bool) {
	id, ok := c.Locals(localPortalSessionID).(uuid.UUID)
	return id, ok
}

func PortalCSRFHash(c *fiber.Ctx) ([]byte, bool) {
	hash, ok := c.Locals(localPortalCSRFHash).([]byte)
	return hash, ok
}

func EmpleadorID(c *fiber.Ctx) (uuid.UUID, bool) {
	id, ok := c.Locals(localEmpleadorID).(uuid.UUID)
	return id, ok
}
func EmpleadorSessionID(c *fiber.Ctx) (uuid.UUID, bool) {
	id, ok := c.Locals(localEmpleadorSesionID).(uuid.UUID)
	return id, ok
}
func EmpleadorCSRFHash(c *fiber.Ctx) ([]byte, bool) {
	hash, ok := c.Locals(localEmpleadorCSRFHash).([]byte)
	return hash, ok
}
