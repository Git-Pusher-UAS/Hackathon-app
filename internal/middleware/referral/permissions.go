// Package referral agrupa los permisos de /referrals (§17).
package referral

import (
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

const (
	// PermRead ve la bandeja de canalizaciones de sus áreas.
	PermRead = "referral.read"
	// PermTransition acepta, rechaza, completa o cancela canalizaciones.
	PermTransition = "referral.transition"
)

var atencion = []string{"TrabajoSocial", "Psicología", "Jurídico"}

// Permisos los usa el seed.
var Permisos = []middleware.PermisoSeed{
	{Codigo: PermRead, Descripcion: "Ver la bandeja de canalizaciones de sus áreas", Roles: atencion},
	{Codigo: PermTransition, Descripcion: "Aceptar, rechazar, completar o cancelar canalizaciones", Roles: atencion},
}

// Require exige un permiso del token (después de AuthMiddleware).
func Require(codigo string) fiber.Handler { return middleware.RequirePermiso(codigo) }
