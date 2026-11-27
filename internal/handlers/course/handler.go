// Package course expone /api/v1/courses.
package course

import (
	"errors"
	"log"
	"time"

	hu "backend-hackathon/internal/handlers/handlerutil"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	cmodels "backend-hackathon/internal/models/course"
	svc "backend-hackathon/internal/services/course"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ cmodels.Page

// MapError traduce los errores del módulo al catálogo §18.
func MapError(err error) error {
	var he *httpx.Error
	switch {
	case errors.As(err, &he):
		return he
	case errors.Is(err, svc.ErrNotFound):
		return httpx.E(fiber.StatusNotFound, "not_found", "recurso no disponible")
	default:
		log.Printf("course: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "internal_error", "error interno")
	}
}

func dateQuery(c *fiber.Ctx, name string) (*time.Time, error) {
	raw := c.Query(name)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, httpx.Invalid(name+" debe ser AAAA-MM-DD", map[string]string{name: "formato_invalido"})
	}
	return &t, nil
}

// List devuelve los cursos, charlas y conferencias.
//
// @Summary      Listar cursos
// @Description  Eventos de Capacitación, de la fecha más próxima a la más lejana, con cupo e inscritas. Requiere courses.manage.
// @Tags         courses
// @Produce      json
// @Security     BearerAuth
// @Param        type    query  string  false  "curso | charla | conferencia"
// @Param        from    query  string  false  "Fecha desde (AAAA-MM-DD, inclusive)"
// @Param        to      query  string  false  "Fecha hasta (AAAA-MM-DD, inclusive)"
// @Param        limit   query  int     false  "1..100 (20 por defecto)"
// @Param        cursor  query  string  false  "Cursor opaco"
// @Success      200  {object}  cmodels.Page
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Router       /courses [get]
func (h *Handler) List(c *fiber.Ctx) error {
	actor, err := hu.Actor(c)
	if err != nil {
		return err
	}
	var q svc.ListQuery
	if raw := c.Query("type"); raw != "" {
		t := models.TipoCurso(raw)
		if !svc.ValidTipo(t) {
			return httpx.Invalid("type inválido", map[string]string{"type": "valor_invalido"})
		}
		q.Tipo = &t
	}
	if q.Desde, err = dateQuery(c, "from"); err != nil {
		return err
	}
	if q.Hasta, err = dateQuery(c, "to"); err != nil {
		return err
	}
	page, err := hu.Page(c)
	if err != nil {
		return err
	}
	res, err := h.service.List(c.Context(), actor, q, page)
	if err != nil {
		return MapError(err)
	}
	return c.JSON(res)
}

// Get devuelve un curso.
//
// @Summary      Obtener curso
// @Description  Requiere courses.manage.
// @Tags         courses
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "UUID del curso"
// @Success      200  {object}  cmodels.Curso
// @Failure      404  {object}  httpx.ErrorResponse
// @Router       /courses/{id} [get]
func (h *Handler) Get(c *fiber.Ctx) error {
	id, err := hu.UUIDParam(c, "id")
	if err != nil {
		return err
	}
	res, err := h.service.Get(c.Context(), id)
	if err != nil {
		return MapError(err)
	}
	return c.JSON(res)
}

// Create publica un curso, charla o conferencia.
//
// @Summary      Crear curso
// @Description  Se publica en el foro de inmediato. La fecha no puede ser pasada (hora de Los Mochis) y el cupo debe ser de 1 o más. Requiere courses.manage.
// @Tags         courses
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-CSRF-Token     header  string                 true   "Token CSRF de la sesión"
// @Param        Idempotency-Key  header  string                 false  "UUID de la operación"
// @Param        request          body    cmodels.CreateRequest  true   "Datos del curso"
// @Success      201  {object}  cmodels.Curso
// @Failure      422  {object}  httpx.ErrorResponse  "validation_failed con fields (fecha_pasada, minimo_1...)"
// @Router       /courses [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	actor, err := hu.Actor(c)
	if err != nil {
		return err
	}
	var req cmodels.CreateRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.Create(c.Context(), actor, req)
	if err != nil {
		return MapError(err)
	}
	c.Set(fiber.HeaderLocation, "/api/v1/courses/"+res.ID.String())
	return c.Status(fiber.StatusCreated).JSON(res)
}
