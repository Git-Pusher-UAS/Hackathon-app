// Package course agrupa los permisos de /courses y /outreach-visits (§17).
package course

import (
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

const (
	// PermManage crea cursos, charlas y conferencias y pasa lista.
	PermManage = "courses.manage"
	// PermOutreach planea y consulta visitas a sectores.
	PermOutreach = "outreach.manage"
)

// Permisos los usa el seed.
var Permisos = []middleware.PermisoSeed{
	{Codigo: PermManage, Descripcion: "Crear cursos, charlas y conferencias y pasar lista", Roles: []string{"Capacitación"}},
	{Codigo: PermOutreach, Descripcion: "Planear y consultar visitas a sectores", Roles: []string{"Capacitación"}},
}

// Require exige un permiso del token (después de AuthMiddleware).
func Require(codigo string) fiber.Handler { return middleware.RequirePermiso(codigo) }
