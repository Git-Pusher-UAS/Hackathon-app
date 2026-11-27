// Package analytics agrupa los permisos de /analytics (§17).
package analytics

import (
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// PermRead consulta los indicadores agregados.
const PermRead = "analytics.read"

// PermHeatmap consulta el mapa de calor por AGEB.
const PermHeatmap = "analytics.heatmap"

// Permisos los usa el seed.
var Permisos = []middleware.PermisoSeed{
	{Codigo: PermRead, Descripcion: "Consultar indicadores agregados", Roles: []string{"Dirección", "AdministraciónAnalítica"}},
	{Codigo: PermHeatmap, Descripcion: "Consultar el mapa de calor por AGEB", Roles: []string{"Capacitación", "Dirección", "AdministraciónAnalítica"}},
}

// Require exige un permiso del token (después de AuthMiddleware).
func Require(codigo string) fiber.Handler { return middleware.RequirePermiso(codigo) }
