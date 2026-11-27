package outreach

import (
	"errors"
	"log"
	"strconv"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	"backend-hackathon/internal/models"
	omodels "backend-hackathon/internal/models/outreach"
	repository "backend-hackathon/internal/repository/repositories"
	svc "backend-hackathon/internal/services/outreach"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

func mapError(err error) error {
	var he *httpx.Error
	if errors.As(err, &he) {
		return he
	}
	log.Printf("outreach: %v", err)
	return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
}

// List lista visitas de vinculación.
//
// @Summary      Listar visitas de vinculación
// @Description  Devuelve visitas de vinculación comunitaria.
// @Tags         outreach-visits
// @Produce      json
// @Security     BearerAuth
// @Param        status  query  string  false  "planeada | realizada"
// @Param        limit   query  int     false  "1..100"
// @Param        cursor  query  string  false  "Cursor opaco"
// @Success      200  {array}   omodels.Visita
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /outreach-visits [get]
func (h *Handler) List(c *fiber.Ctx) error {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	var estado *models.EstadoVisita
	if raw := c.Query("status"); raw != "" {
		st := models.EstadoVisita(raw)
		if st != models.VisitaPlaneada && st != models.VisitaRealizada {
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "status invalido")
		}
		estado = &st
	}
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "limit invalido")
		}
		limit = n
	}
	res, err := h.service.List(c.Context(), actor, estado, repository.PageRequest{Limit: limit, Cursor: c.Query("cursor")})
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}

// Create registra una visita de vinculación.
//
// @Summary      Crear visita de vinculación
// @Description  Registra una visita con sector, fecha y tema.
// @Tags         outreach-visits
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-CSRF-Token  header  string                  true  "Token CSRF"
// @Param        request       body    omodels.CreateRequest   true  "Datos de la visita"
// @Success      201  {object}  omodels.Visita
// @Failure      400  {object}  httpx.ErrorResponse
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /outreach-visits [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
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
