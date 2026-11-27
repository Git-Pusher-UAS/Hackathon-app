// Package outreach expone /api/v1/outreach-visits.
package outreach

import (
	"errors"
	"log"

	hu "backend-hackathon/internal/handlers/handlerutil"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	omodels "backend-hackathon/internal/models/outreach"
	svc "backend-hackathon/internal/services/outreach"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ omodels.Page

func mapError(err error) error {
	var he *httpx.Error
	if errors.As(err, &he) {
		return he
	}
	log.Printf("outreach: %v", err)
	return httpx.E(fiber.StatusInternalServerError, "internal_error", "error interno")
}

// List devuelve las visitas a sectores.
//
// @Summary      Listar visitas a sectores
// @Description  Visitas de Capacitación de la fecha más próxima a la más lejana. Requiere outreach.manage.
// @Tags         outreach-visits
// @Produce      json
// @Security     BearerAuth
// @Param        status  query  string  false  "planeada | realizada"
// @Param        limit   query  int     false  "1..100 (20 por defecto)"
// @Param        cursor  query  string  false  "Cursor opaco"
// @Success      200  {object}  omodels.Page
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Router       /outreach-visits [get]
func (h *Handler) List(c *fiber.Ctx) error {
	actor, err := hu.Actor(c)
	if err != nil {
		return err
	}
	var estado *models.EstadoVisita
	if raw := c.Query("status"); raw != "" {
		st := models.EstadoVisita(raw)
		if st != models.VisitaPlaneada && st != models.VisitaRealizada {
			return httpx.Invalid("status inválido", map[string]string{"status": "valor_invalido"})
		}
		estado = &st
	}
	page, err := hu.Page(c)
	if err != nil {
		return err
	}
	res, err := h.service.List(c.Context(), actor, estado, page)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}

// Create planea una visita a un sector.
//
// @Summary      Planear visita a un sector
// @Description  Nace en estado planeada; el responsable es el área de quien la crea. Requiere outreach.manage.
// @Tags         outreach-visits
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-CSRF-Token     header  string                 true   "Token CSRF de la sesión"
// @Param        Idempotency-Key  header  string                 false  "UUID de la operación"
// @Param        request          body    omodels.CreateRequest  true   "Sector, fecha y tema"
// @Success      201  {object}  omodels.Visita
// @Failure      422  {object}  httpx.ErrorResponse
// @Router       /outreach-visits [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	actor, err := hu.Actor(c)
	if err != nil {
		return err
	}
	var req omodels.CreateRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.Create(c.Context(), actor, req)
	if err != nil {
		return mapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(res)
}
