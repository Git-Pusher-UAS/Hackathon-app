// Package institutionalcontact expone GET /api/v1/institutional-contacts.
package institutionalcontact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"

	"backend-hackathon/internal/httpx"
	icmodels "backend-hackathon/internal/models/institutionalcontact"
	svc "backend-hackathon/internal/services/institutionalcontact"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ icmodels.Contact

// List devuelve el directorio institucional público.
//
// @Summary      Directorio institucional
// @Description  Contactos de ayuda publicables, primero los prioritarios. Público (sin sesión). Arreglo directo, cacheable 1 h con ETag.
// @Tags         institutional-contacts
// @Produce      json
// @Param        If-None-Match  header  string  false  "ETag de una respuesta anterior"
// @Success      200  {array}   icmodels.Contact
// @Success      304  "Sin cambios"
// @Header       200  {string}  ETag  "Versión del directorio"
// @Router       /institutional-contacts [get]
func (h *Handler) List(c *fiber.Ctx) error {
	res, err := h.service.List(c.Context())
	if err != nil {
		log.Printf("institutionalcontact: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "internal_error", "error interno")
	}
	body, err := json.Marshal(res)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(body)
	etag := httpx.FormatETag(hex.EncodeToString(sum[:8]))
	c.Set(fiber.HeaderETag, etag)
	c.Set(fiber.HeaderCacheControl, "public, max-age=3600")
	if c.Get(fiber.HeaderIfNoneMatch) == etag {
		return c.SendStatus(fiber.StatusNotModified)
	}
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
	return c.Send(body)
}
