// Package httpx agrupa helpers de respuesta HTTP.
package httpx

import (
	"log"

	"github.com/gofiber/fiber/v3"
)

// ErrorBody es el envelope estándar de error de la API.
type ErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// JSON escribe v como JSON con el status, charset y salto final existentes.
func JSON(c fiber.Ctx, status int, v any) error {
	c.Status(status)
	if v == nil {
		c.Set(fiber.HeaderContentType, "application/json; charset=utf-8")
		return c.Send(nil)
	}
	if err := c.JSON(v, "application/json; charset=utf-8"); err != nil {
		log.Printf("httpx: no se pudo escribir la respuesta: %v", err)
		return err
	}
	return c.Send(append(c.Response().Body(), '\n'))
}

// Error escribe un envelope de error.
func Error(c fiber.Ctx, status int, code, message string) error {
	return JSON(c, status, ErrorBody{Error: code, Message: message})
}
