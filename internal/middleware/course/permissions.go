package course

import (
	"backend-hackathon/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

const PermManage = "courses.manage"

func Require(codigo string) fiber.Handler { return middleware.RequirePermiso(codigo) }
