// Package referral define los DTOs de la bandeja de canalizaciones (API-032)
// y de sus transiciones (API-034).
package referral

import (
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

// Direcciones de la bandeja.
const (
	DirectionIncoming = "incoming" // el departamento del actor es el destino
	DirectionOutgoing = "outgoing" // el departamento del actor es el origen
)

// Acciones de transición.
const (
	ActionAccept   = "accept"
	ActionReject   = "reject"
	ActionComplete = "complete"
	ActionCancel   = "cancel"
)

// Ref es un departamento con nombre visible.
type Ref struct {
	ID     uuid.UUID `json:"id"`
	Nombre string    `json:"nombre"`
}

// View es una canalización interna.
type View struct {
	ID           uuid.UUID                 `json:"id"`
	ExpedienteID uuid.UUID                 `json:"expedienteId"`
	Alias        string                    `json:"alias"`
	Origen       Ref                       `json:"origen"`
	Destino      Ref                       `json:"destino"`
	Resumen      string                    `json:"resumen"`
	Estado       models.EstadoCanalizacion `json:"estado"`
	VersionTag   string                    `json:"versionTag"`
	CreatedAt    time.Time                 `json:"createdAt"`
	UpdatedAt    time.Time                 `json:"updatedAt"`
}

// Page es la página de la bandeja.
type Page struct {
	Items      []View  `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

// ListQuery son los filtros de la bandeja.
type ListQuery struct {
	Direction string
	Estados   []models.EstadoCanalizacion
	RecordID  *uuid.UUID
}

// TransitionRequest cambia el estado. Motivo es obligatorio en reject y cancel.
//
//	accept:   pendiente → aceptada   (departamento destino; asigna al actor)
//	reject:   pendiente → rechazada  (departamento destino)
//	complete: aceptada  → completada (departamento destino)
//	cancel:   pendiente → cancelada  (departamento origen)
type TransitionRequest struct {
	Action string  `json:"action" example:"accept" enums:"accept,reject,complete,cancel"`
	Motivo *string `json:"motivo,omitempty"`
}
