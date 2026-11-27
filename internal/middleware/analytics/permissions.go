// Package analytics agrupa los permisos de /analytics (§17).
package analytics

import (
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// PermRead consulta los indicadores agregados.
const PermRead = "analytics.read"

// Permisos los usa el seed.
var Permisos = []middleware.PermisoSeed{
	{Codigo: PermRead, Descripcion: "Consultar indicadores agregados", Roles: []string{"Dirección", "AdministraciónAnalítica"}},
}

// Require exige un permiso del token (después de AuthMiddleware).
func Require(codigo string) fiber.Handler { return middleware.RequirePermiso(codigo) }
