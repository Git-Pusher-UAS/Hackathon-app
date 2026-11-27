package survey

import (
	"backend-hackathon/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

const PermResults = "surveys.results"

func Require(codigo string) fiber.Handler { return middleware.RequirePermiso(codigo) }
