// Package handlerutil agrupa la lectura de peticiones que comparten los
// handlers del contrato: actor autenticado, parámetros UUID, X-Operation-Id,
// paginación y fechas RFC 3339.
package handlerutil

import (
	"strconv"
	"strings"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Actor devuelve el actor que dejó AuthMiddleware.
func Actor(c *fiber.Ctx) (repository.ActorContext, error) {
	a, ok := middleware.ActorFrom(c)
	if !ok {
		return a, httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesión inválida")
	}
	return a, nil
}

// UUIDParam lee un parámetro de ruta como UUID.
func UUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(name))
	if err != nil {
		return uuid.Nil, httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", name+" inválido")
	}
	return id, nil
}

// UUIDQuery lee un parámetro de consulta opcional como UUID.
func UUIDQuery(c *fiber.Ctx, name string) (*uuid.UUID, error) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", name+" inválido")
	}
	return &id, nil
}

// OperationID lee X-Operation-Id (UUID generado por el cliente), que hace
// idempotentes las escrituras.
func OperationID(c *fiber.Ctx) (uuid.UUID, error) {
	raw := strings.TrimSpace(c.Get("X-Operation-Id"))
	if raw == "" {
		return uuid.Nil, httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "X-Operation-Id requerido (UUID)")
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "X-Operation-Id inválido")
	}
	return id, nil
}

// Page lee limit (1..100, 20 por defecto) y cursor.
func Page(c *fiber.Ctx) (repository.PageRequest, error) {
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			return repository.PageRequest{}, httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "limit debe estar entre 1 y 100")
		}
		limit = n
	}
	return repository.PageRequest{Limit: limit, Cursor: c.Query("cursor")}, nil
}

// TimeQuery lee un parámetro de consulta opcional en RFC 3339.
func TimeQuery(c *fiber.Ctx, name string) (*time.Time, error) {
	raw := c.Query(name)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", name+" debe ser RFC3339")
	}
	return &t, nil
}

// Range arma un TimeRange con from/to opcionales.
func Range(c *fiber.Ctx) (repository.TimeRange, error) {
	var r repository.TimeRange
	from, err := TimeQuery(c, "from")
	if err != nil {
		return r, err
	}
	to, err := TimeQuery(c, "to")
	if err != nil {
		return r, err
	}
	if from != nil {
		r.Desde = *from
	}
	if to != nil {
		r.Hasta = *to
	}
	if from != nil && to != nil && !to.After(*from) {
		return r, httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "to debe ser posterior a from")
	}
	return r, nil
}
