// Package auditlog agrupa el permiso de /audit-logs (§17).
package auditlog

import (
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// PermRead consulta la bitácora de actividad.
const PermRead = "auditlog.read"

// Permisos los usa el seed.
var Permisos = []middleware.PermisoSeed{
	{Codigo: PermRead, Descripcion: "Consultar la bitácora de actividad", Roles: []string{"Dirección", "AdministradorTI"}},
}

// Require exige PermRead (después de AuthMiddleware).
func Require() fiber.Handler { return middleware.RequirePermiso(PermRead) }
