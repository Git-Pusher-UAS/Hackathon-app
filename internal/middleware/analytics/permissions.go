package analytics

import (
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

const (
	PermRead    = "analytics.read"
	PermHeatmap = "analytics.heatmap"
)

func Require(codigo string) fiber.Handler { return middleware.RequirePermiso(codigo) }
