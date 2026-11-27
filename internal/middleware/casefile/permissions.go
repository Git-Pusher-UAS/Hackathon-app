// Package casefile agrupa middleware de las rutas /api/case-files.
package casefile

import (
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

const (
	PermRead     = "casefile.read"
	PermWrite    = "casefile.write"
	PermAssign   = "casefile.assign"
	PermDisclose = "casefile.disclose"
	PermConsent  = "casefile.consent"
	PermRefer    = "casefile.refer"
)

type Permiso struct{ Codigo, Descripcion string }

// Permisos lo usa el seed para crearlos en la BD.
var Permisos = []Permiso{
	{PermRead, "Consultar expedientes asignados"},
	{PermWrite, "Crear y editar expedientes y notas"},
	{PermAssign, "Asignar y revocar personal en expedientes"},
	{PermDisclose, "Revelar campos sensibles de la persona"},
	{PermConsent, "Registrar y revocar consentimientos"},
	{PermRefer, "Crear canalizaciones internas"},
}

// Require exige un permiso del token (despu\u00e9s de AuthMiddleware).
func Require(codigo string) fiber.Handler { return middleware.RequirePermiso(codigo) }
