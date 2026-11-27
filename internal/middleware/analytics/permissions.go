// Package analytics agrupa el permiso de /analytics.
package analytics

import (
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// PermRead consulta los agregados del tablero (con supresión k = 5).
const PermRead = "analytics.read"

// Descripcion la usa el seed.
const Descripcion = "Consultar indicadores agregados y mapa de calor"

// Require exige PermRead (después de AuthMiddleware).
func Require() fiber.Handler { return middleware.RequirePermiso(PermRead) }
