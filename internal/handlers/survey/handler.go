// Package survey expone GET /api/v1/surveys/results.
package survey

import (
	"log"

	"backend-hackathon/internal/httpx"
	smodels "backend-hackathon/internal/models/survey"
	svc "backend-hackathon/internal/services/survey"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ smodels.Resultado

// Results devuelve los resultados agregados de las encuestas.
//
// @Summary      Resultados de encuestas
// @Description  Arreglo directo, solo agregados: promedio de las preguntas de escala y conteo por opción. Cualquier cifra con menos de 5 respuestas llega en null. Las respuestas abiertas no se devuelven. Requiere surveys.results.
// @Tags         surveys
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   smodels.Resultado
// @Failure      403  {object}  httpx.ErrorResponse
// @Router       /surveys/results [get]
func (h *Handler) Results(c *fiber.Ctx) error {
	res, err := h.service.Results(c.Context())
	if err != nil {
		log.Printf("survey: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "internal_error", "error interno")
	}
	return c.JSON(res)
}
