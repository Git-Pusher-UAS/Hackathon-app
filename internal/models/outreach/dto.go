package outreach

import (
	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

type Visita struct {
	ID          uuid.UUID           `json:"id"`
	Sector      string              `json:"sector"`
	Fecha       string              `json:"fecha"`
	Tema        string              `json:"tema"`
	Responsable string              `json:"responsable"`
	Estado      models.EstadoVisita `json:"estado"`
}

type CreateRequest struct {
	Sector string `json:"sector"`
	Fecha  string `json:"fecha"`
	Tema   string `json:"tema"`
}
