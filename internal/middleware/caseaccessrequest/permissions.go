// Package caseaccessrequest agrupa los permisos de /case-access-requests.
package caseaccessrequest

import (
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

const (
	PermRequest = "caseaccess.request"
	PermReview  = "caseaccess.review"
)

func Require(codigo string) fiber.Handler { return middleware.RequirePermiso(codigo) }
