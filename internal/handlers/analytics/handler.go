// Package analytics expone GET /analytics/summary y GET /analytics/heatmap.
package analytics

import (
	"log"
	"time"

	hu "backend-hackathon/internal/handlers/handlerutil"
	"backend-hackathon/internal/httpx"
	anmodels "backend-hackathon/internal/models/analytics"
	repository "backend-hackathon/internal/repository/repositories"
	svc "backend-hackathon/internal/services/analytics"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

var _ anmodels.Summary

func internal(err error) error {
	log.Printf("analytics: %v", err)
	return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
}

// Summary devuelve los agregados por sector, tipo de caso y mes.
//
// @Summary      Resumen de indicadores
// @Description  Personas y episodios por sector, tipo de caso y mes. Las celdas con menos de 5 casos llegan en null con suprimido=true. Requiere analytics.read. API-037.
// @Tags         analytics
// @Produce      json
// @Security     BearerAuth
// @Param        from        query  string  false  "Desde (RFC 3339), sobre el mes del episodio"
// @Param        to          query  string  false  "Hasta (RFC 3339, exclusivo)"
// @Param        sectorId    query  string  false  "UUID del sector"
// @Param        caseTypeId  query  string  false  "UUID del tipo de caso"
// @Success      200  {object}  anmodels.Summary
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /analytics/summary [get]
func (h *Handler) Summary(c *fiber.Ctx) error {
	var f repository.ResumenFiltro
	var err error
	if f.Rango, err = hu.Range(c); err != nil {
		return err
	}
	if f.SectorID, err = hu.UUIDQuery(c, "sectorId"); err != nil {
		return err
	}
	if f.TipoCasoID, err = hu.UUIDQuery(c, "caseTypeId"); err != nil {
		return err
	}
	res, err := h.service.Summary(c.Context(), f)
	if err != nil {
		return internal(err)
	}
	return c.JSON(res)
}

// Heatmap devuelve el nivel por sector.
//
// @Summary      Mapa de calor por sector
// @Description  Nivel (bajo, medio, alto) y conteo por sector, tipo de caso y mes, sin puntos domiciliarios. Sin filtros puede haber varias filas por sector (una por mes y tipo). Requiere analytics.read. API-038.
// @Tags         analytics
// @Produce      json
// @Security     BearerAuth
// @Param        period      query  string  false  "Mes en formato YYYY-MM"
// @Param        caseTypeId  query  string  false  "UUID del tipo de caso"
// @Success      200  {object}  anmodels.Heatmap
// @Failure      401  {object}  httpx.ErrorResponse
// @Failure      403  {object}  httpx.ErrorResponse
// @Failure      422  {object}  httpx.ErrorResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /analytics/heatmap [get]
func (h *Handler) Heatmap(c *fiber.Ctx) error {
	var f repository.MapaFiltro
	var err error
	if f.TipoCasoID, err = hu.UUIDQuery(c, "caseTypeId"); err != nil {
		return err
	}
	if raw := c.Query("period"); raw != "" {
		p, err := time.Parse("2006-01", raw)
		if err != nil {
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "period debe ser YYYY-MM")
		}
		f.Periodo = &p
	}
	res, err := h.service.Heatmap(c.Context(), f)
	if err != nil {
		return internal(err)
	}
	return c.JSON(res)
}
