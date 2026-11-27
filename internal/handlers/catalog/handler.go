// Package catalog expone GET /catalogs y GET /institutional-contacts.
package catalog

import (
	"log"

	"backend-hackathon/internal/httpx"
	catmodels "backend-hackathon/internal/models/catalog"
	contactmodels "backend-hackathon/internal/models/institutionalcontact"
	svc "backend-hackathon/internal/services/catalog"

	"github.com/gofiber/fiber/v2"
)

// Referencias para que swag resuelva los tipos de las anotaciones.
var (
	_ catmodels.Response
	_ contactmodels.Response
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

func internal(err error) error {
	log.Printf("catalog: %v", err)
	return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
}

// Catalogs devuelve los catálogos activos para filtros y formularios.
//
// @Summary      Catálogos
// @Description  Departamentos, roles, sectores (sin geometría), tipos de caso, orígenes de contacto, finalidades de consentimiento e instituciones activas. API-015.
// @Tags         catalogs
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  catmodels.Response
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /catalogs [get]
func (h *Handler) Catalogs(c *fiber.Ctx) error {
	res, err := h.service.Catalogs(c.Context())
	if err != nil {
		return internal(err)
	}
	return c.JSON(res)
}

// Contacts devuelve el directorio institucional público.
//
// @Summary      Directorio institucional
// @Description  Instituciones publicables con teléfono y horario. Es público (no requiere sesión). `version` cambia cuando cambia algún contacto y se repite en el ETag. API-039.
// @Tags         institutional-contacts
// @Produce      json
// @Success      200  {object}  contactmodels.Response
// @Header       200  {string}  ETag  "Versión del directorio"
// @Success      304  "Sin cambios (If-None-Match)"
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /institutional-contacts [get]
func (h *Handler) Contacts(c *fiber.Ctx) error {
	res, err := h.service.Contacts(c.Context())
	if err != nil {
		return internal(err)
	}
	etag := httpx.FormatETag(res.Version)
	c.Set(fiber.HeaderETag, etag)
	c.Set(fiber.HeaderCacheControl, "public, max-age=300")
	if c.Get(fiber.HeaderIfNoneMatch) == etag {
		return c.SendStatus(fiber.StatusNotModified)
	}
	return c.JSON(res)
}
