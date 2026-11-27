// Package analytics define los DTOs de GET /analytics/summary (API-037) y
// GET /analytics/heatmap (API-038). Un número nil significa celda suprimida
// (menos de 5 casos): el cliente lo muestra como "<5", nunca como 0.
package analytics

import (
	"time"

	"github.com/google/uuid"
)

// SupresionK es el mínimo de casos para publicar una celda.
const SupresionK = 5

// SummaryRow es un agregado por sector, tipo de caso y mes.
type SummaryRow struct {
	SectorID   *uuid.UUID `json:"sectorId"`
	TipoCasoID *uuid.UUID `json:"tipoCasoId"`
	Periodo    time.Time  `json:"periodo"` // primer día del mes
	Personas   *int       `json:"personas"`
	Episodios  *int       `json:"episodios"`
	Suprimido  bool       `json:"suprimido"`
}

// Summary es la respuesta del resumen.
type Summary struct {
	GeneradoEn time.Time    `json:"generadoEn"`
	SupresionK int          `json:"supresionK" example:"5"`
	Items      []SummaryRow `json:"items"`
}

// HeatmapCell es el nivel publicable de un sector.
type HeatmapCell struct {
	SectorID   uuid.UUID  `json:"sectorId"`
	TipoCasoID *uuid.UUID `json:"tipoCasoId"`
	Periodo    time.Time  `json:"periodo"`
	Nivel      *string    `json:"nivel" example:"medio"` // bajo | medio | alto
	Conteo     *int       `json:"conteo"`
	Suprimido  bool       `json:"suprimido"`
}

// Heatmap es la respuesta del mapa de calor. No contiene puntos domiciliarios.
type Heatmap struct {
	GeneradoEn time.Time     `json:"generadoEn"`
	SupresionK int           `json:"supresionK" example:"5"`
	Items      []HeatmapCell `json:"items"`
}
