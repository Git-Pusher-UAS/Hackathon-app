// Package survey agrupa el permiso de /surveys/results (§17).
package survey

import (
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// PermResults consulta los resultados agregados de las encuestas.
const PermResults = "surveys.results"

// Permisos los usa el seed.
var Permisos = []middleware.PermisoSeed{
	{Codigo: PermResults, Descripcion: "Consultar resultados agregados de encuestas", Roles: []string{"AdministraciónAnalítica"}},
}

// Require exige PermResults (después de AuthMiddleware).
func Require() fiber.Handler { return middleware.RequirePermiso(PermResults) }
