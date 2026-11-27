package outreach

import (
	"backend-hackathon/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

const PermManage = "outreach.manage"

func Require(codigo string) fiber.Handler { return middleware.RequirePermiso(codigo) }
