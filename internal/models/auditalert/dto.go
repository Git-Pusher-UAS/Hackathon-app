package auditalert

import (
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

type Item struct {
	ID           uuid.UUID           `json:"id"`
	Estado       models.EstadoAlerta `json:"estado"`
	Fecha        time.Time           `json:"fecha"`
	ExpedienteID *uuid.UUID          `json:"expedienteId"`
}
