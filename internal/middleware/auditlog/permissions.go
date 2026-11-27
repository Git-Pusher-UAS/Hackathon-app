// Package auditlog agrupa el permiso de /audit-logs y /audit-alerts.
package auditlog

import (
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// PermRead consulta la bitácora y las alertas de acceso.
const PermRead = "audit.read"

// Descripcion la usa el seed.
const Descripcion = "Consultar la bitácora de auditoría y las alertas de acceso"

// Require exige PermRead (después de AuthMiddleware).
func Require() fiber.Handler { return middleware.RequirePermiso(PermRead) }
