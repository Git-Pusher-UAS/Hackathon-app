package httpx

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/gofiber/fiber/v2"
)

// DecodeJSON decodifica el cuerpo rechazando campos desconocidos y contenido
// extra después del primer valor JSON.
func DecodeJSON(c *fiber.Ctx, dst any) error {
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return E(fiber.StatusBadRequest, "invalid_json", "cuerpo de la petición inválido")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return E(fiber.StatusBadRequest, "invalid_json", "cuerpo de la petición inválido")
	}
	return nil
}
