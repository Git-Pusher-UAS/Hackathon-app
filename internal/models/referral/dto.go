package referral

import (
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

const (
	DirectionIncoming = "incoming"
	DirectionOutgoing = "outgoing"
)

type Origen struct {
	DepartamentoID uuid.UUID `json:"departamentoId"`
	Nombre         string    `json:"nombre"`
}

type Destino struct {
	DepartamentoID *uuid.UUID `json:"departamentoId,omitempty"`
	InstitucionID  *uuid.UUID `json:"institucionId,omitempty"`
	Nombre         string     `json:"nombre"`
}

type View struct {
	ID           uuid.UUID                 `json:"id"`
	ExpedienteID uuid.UUID                 `json:"expedienteId"`
	Alias        string                    `json:"alias"`
	Tipo         models.TipoCanalizacion   `json:"tipo"`
	Origen       Origen                    `json:"origen"`
	Destino      Destino                   `json:"destino"`
	Resumen      string                    `json:"resumen"`
	Estado       models.EstadoCanalizacion `json:"estado"`
	Fecha        time.Time                 `json:"fecha"`
	VersionTag   string                    `json:"versionTag"`
}

type TransitionRequest struct {
	Action string  `json:"action"`
	Motivo *string `json:"motivo,omitempty"`
}

type ListQuery struct {
	Direction string
	Estados   []models.EstadoCanalizacion
	RecordID  *uuid.UUID
	RecordRef string
}
