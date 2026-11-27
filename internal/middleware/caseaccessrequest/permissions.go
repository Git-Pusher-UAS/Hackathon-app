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

// Permiso es un permiso que crea el seed con los roles que lo reciben.
type Permiso struct {
	Codigo, Descripcion string
	Roles               []string
}

// Permisos los usa el seed.
var Permisos = []Permiso{
	{PermRequest, "Pedir acceso a un expediente que lleva otra área", []string{"TrabajoSocial", "Psicología", "Jurídico"}},
	{PermReview, "Revisar y resolver solicitudes de acceso a expedientes", []string{"Coordinación"}},
}

// Require exige un permiso del token (después de AuthMiddleware).
func Require(codigo string) fiber.Handler { return middleware.RequirePermiso(codigo) }
