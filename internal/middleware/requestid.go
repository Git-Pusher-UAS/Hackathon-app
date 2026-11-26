package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/gofiber/fiber/v3"
)

type ctxKey string

const (
	// RequestIDKey es la clave de contexto donde se guarda el request id.
	RequestIDKey ctxKey = "request_id"
	// HeaderRequestID es el header de correlación.
	HeaderRequestID = "X-Request-ID"
)

// RequestID propaga o genera un identificador de petición.
// Se conserva la propagación literal: el middleware oficial de Fiber sustituye
// los IDs con caracteres no ASCII que la API anterior aceptaba.
func RequestID(c fiber.Ctx) error {
	id := c.Get(HeaderRequestID)
	if id == "" {
		id = newRequestID()
	}
	c.Set(HeaderRequestID, id)
	ctx := context.WithValue(c.Context(), RequestIDKey, id)
	c.SetContext(ctx)
	return c.Next()
}

// RequestIDFrom devuelve el request id del contexto, si existe.
func RequestIDFrom(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
