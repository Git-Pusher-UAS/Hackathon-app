// Package catalog expone GET /api/v1/catalogs.
package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"

	"backend-hackathon/internal/httpx"
	catmodels "backend-hackathon/internal/models/catalog"
	svc "backend-hackathon/internal/services/catalog"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ catmodels.Response

// Catalogs devuelve los catálogos activos.
//
// @Summary      Catálogos
// @Description  Una sola llamada al entrar: departamentos, sectores (sin geometría), orígenes de contacto, instituciones, tipos de caso, finalidades del consentimiento, motivos para revelar datos y el vocabulario de la bolsa de trabajo. Cacheable: ETag y Cache-Control private, max-age=3600.
// @Tags         catalogs
// @Produce      json
// @Security     BearerAuth
// @Param        If-None-Match  header  string  false  "ETag de una respuesta anterior"
// @Success      200  {object}  catmodels.Response
// @Success      304  "Sin cambios"
// @Header       200  {string}  ETag  "Versión de los catálogos"
// @Failure      401  {object}  httpx.ErrorResponse
// @Router       /catalogs [get]
func (h *Handler) Catalogs(c *fiber.Ctx) error {
	res, err := h.service.Catalogs(c.Context())
	if err != nil {
		log.Printf("catalog: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "internal_error", "error interno")
	}
	body, err := json.Marshal(res)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(body)
	etag := httpx.FormatETag(hex.EncodeToString(sum[:8]))
	c.Set(fiber.HeaderETag, etag)
	c.Set(fiber.HeaderCacheControl, "private, max-age=3600")
	if c.Get(fiber.HeaderIfNoneMatch) == etag {
		return c.SendStatus(fiber.StatusNotModified)
	}
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
	return c.Send(body)
}
