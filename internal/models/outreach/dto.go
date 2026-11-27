// Package outreach define los DTOs de /outreach-visits (§11.6).
package outreach

import (
	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

// Visita es una visita a un sector.
type Visita struct {
	ID          uuid.UUID           `json:"id"`
	SectorID    uuid.UUID           `json:"sectorId"`
	Sector      string              `json:"sector" example:"Sector 12"`
	Fecha       string              `json:"fecha" example:"2026-12-03"`
	Tema        string              `json:"tema" example:"Prevención de la violencia en el noviazgo"`
	Responsable string              `json:"responsable" example:"Capacitación"`
	Estado      models.EstadoVisita `json:"estado" enums:"planeada,realizada"`
}

// CreateRequest es el cuerpo de POST /outreach-visits.
type CreateRequest struct {
	SectorID string `json:"sectorId"`
	Fecha    string `json:"fecha" example:"2026-12-03"`
	Tema     string `json:"tema"`
}

// Page es la página de visitas.
type Page struct {
	Items      []Visita `json:"items"`
	NextCursor *string  `json:"nextCursor"`
}
