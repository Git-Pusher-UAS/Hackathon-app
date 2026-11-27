// Package httpx agrupa utilidades HTTP compartidas por los handlers de /api/v1:
// sobre de errores, ETags/versionTag, cursores firmados y decodificación JSON.
package httpx

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/google/uuid"
)

// Error es un error HTTP con código de contrato.
type Error struct {
	Status  int
	Code    string
	Message string
	// Fields lleva un código por campo inválido (solo en 422).
	Fields map[string]string
	// RetryAfter, en segundos, se envía como encabezado Retry-After (429).
	RetryAfter int
}

func (e *Error) Error() string { return e.Message }

// E construye un Error HTTP.
func E(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// Invalid construye un 422 validation_failed con el código de cada campo.
func Invalid(message string, fields map[string]string) *Error {
	return &Error{Status: fiber.StatusUnprocessableEntity, Code: "validation_failed", Message: message, Fields: fields}
}

// Map traduce cualquier error al sobre de error del contrato /api.
func Map(err error) *Error {
	var httpErr *Error
	if errors.As(err, &httpErr) {
		return httpErr
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return &Error{
			Status:  fiberErr.Code,
			Code:    codeForStatus(fiberErr.Code),
			Message: fiberErr.Message,
		}
	}

	return &Error{Status: fiber.StatusInternalServerError, Code: "internal_error", Message: "error interno"}
}

// Write responde con el sobre de error del contrato.
func Write(c *fiber.Ctx, err error) error {
	mapped := Map(err)
	c.Set(fiber.HeaderCacheControl, "no-store")
	if mapped.RetryAfter > 0 {
		c.Set(fiber.HeaderRetryAfter, strconv.Itoa(mapped.RetryAfter))
	}
	detail := fiber.Map{
		"code":      mapped.Code,
		"message":   mapped.Message,
		"requestId": RequestID(c),
	}
	if len(mapped.Fields) > 0 {
		detail["fields"] = mapped.Fields
	}
	return c.Status(mapped.Status).JSON(fiber.Map{"error": detail})
}

// RequestID devuelve el id de correlación de Fiber o uno nuevo si falta.
func RequestID(c *fiber.Ctx) string {
	raw, _ := c.Locals(requestid.ConfigDefault.ContextKey).(string)
	if raw == "" {
		return uuid.New().String()
	}
	return raw
}

func codeForStatus(status int) string {
	switch status {
	case fiber.StatusBadRequest:
		return "invalid_json"
	case fiber.StatusUnauthorized:
		return "session_expired"
	case fiber.StatusForbidden:
		return "forbidden"
	case fiber.StatusNotFound:
		return "not_found"
	case fiber.StatusMethodNotAllowed:
		return "method_not_allowed"
	case fiber.StatusConflict:
		return "conflict"
	case fiber.StatusGone:
		return "gone"
	case fiber.StatusPreconditionFailed:
		return "precondition_failed"
	case fiber.StatusRequestEntityTooLarge:
		return "payload_too_large"
	case fiber.StatusUnsupportedMediaType:
		return "unsupported_media_type"
	case fiber.StatusUnprocessableEntity:
		return "validation_failed"
	case fiber.StatusPreconditionRequired:
		return "precondition_required"
	case fiber.StatusTooManyRequests:
		return "too_many_attempts"
	case fiber.StatusServiceUnavailable:
		return "encryption_unavailable"
	default:
		return "internal_error"
	}
}
