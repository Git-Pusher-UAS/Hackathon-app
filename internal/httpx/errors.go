// Package httpx agrupa utilidades HTTP compartidas por los handlers de /api:
// sobre de errores, ETags/versionTag, cursores firmados y decodificación JSON.
package httpx

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/google/uuid"
)

// Error es un error HTTP con código de contrato.
type Error struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
}

func (e *Error) Error() string { return e.Message }

// E construye un Error HTTP.
func E(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// Invalid construye un 422 con detalle por campo.
func Invalid(message string, fields map[string]string) *Error {
	return &Error{Status: fiber.StatusUnprocessableEntity, Code: "VALIDATION_ERROR", Message: message, Fields: fields}
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

	return &Error{Status: fiber.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "error interno"}
}

// Write responde con el sobre de error del contrato.
func Write(c *fiber.Ctx, err error) error {
	mapped := Map(err)
	c.Set(fiber.HeaderCacheControl, "no-store")
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
		return "INVALID_JSON"
	case fiber.StatusUnauthorized:
		return "UNAUTHENTICATED"
	case fiber.StatusForbidden:
		return "FORBIDDEN"
	case fiber.StatusNotFound:
		return "RESOURCE_NOT_FOUND"
	case fiber.StatusMethodNotAllowed:
		return "METHOD_NOT_ALLOWED"
	case fiber.StatusConflict:
		return "CONFLICT"
	case fiber.StatusGone:
		return "GONE"
	case fiber.StatusPreconditionFailed:
		return "PRECONDITION_FAILED"
	case fiber.StatusRequestEntityTooLarge:
		return "PAYLOAD_TOO_LARGE"
	case fiber.StatusUnsupportedMediaType:
		return "UNSUPPORTED_MEDIA_TYPE"
	case fiber.StatusUnprocessableEntity:
		return "VALIDATION_ERROR"
	case fiber.StatusPreconditionRequired:
		return "PRECONDITION_REQUIRED"
	case fiber.StatusTooManyRequests:
		return "RATE_LIMITED"
	case fiber.StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	default:
		return "INTERNAL_ERROR"
	}
}
