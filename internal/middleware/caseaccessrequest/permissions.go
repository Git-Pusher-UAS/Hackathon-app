// Package caseaccessrequest agrupa los permisos de /case-access-requests (§17).
package caseaccessrequest

import (
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

const (
	// PermRequest pide acceso a un expediente que lleva otra área.
	PermRequest = "caseaccess.request"
	// PermReview ve la bandeja de solicitudes y las resuelve (Coordinación).
	PermReview = "caseaccess.review"
)

// Permisos los usa el seed.
var Permisos = []middleware.PermisoSeed{
	{Codigo: PermRequest, Descripcion: "Pedir acceso a un expediente que lleva otra área", Roles: []string{"TrabajoSocial", "Psicología", "Jurídico"}},
	{Codigo: PermReview, Descripcion: "Revisar y resolver solicitudes de acceso a expedientes", Roles: []string{"Coordinación"}},
}

// Require exige un permiso del token (después de AuthMiddleware).
func Require(codigo string) fiber.Handler { return middleware.RequirePermiso(codigo) }
