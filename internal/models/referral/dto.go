// Package referral define los DTOs de la bandeja de canalizaciones (§8.2) y
// de sus transiciones (§8.3).
package referral

import (
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

// Direcciones de la bandeja.
const (
	DirectionIncoming = "incoming" // a las áreas del actor
	DirectionOutgoing = "outgoing" // de las áreas del actor
)

// Origen es el departamento que canalizó.
type Origen struct {
	DepartamentoID uuid.UUID `json:"departamentoId"`
	Nombre         string    `json:"nombre"`
}

// Destino es un departamento (interna) o una institución (externa).
type Destino struct {
	DepartamentoID *uuid.UUID `json:"departamentoId"`
	InstitucionID  *uuid.UUID `json:"institucionId"`
	Nombre         string     `json:"nombre"`
}

// View es una canalización.
type View struct {
	ID           uuid.UUID                 `json:"id"`
	ExpedienteID uuid.UUID                 `json:"expedienteId"`
	Alias        string                    `json:"alias"`
	Tipo         models.TipoCanalizacion   `json:"tipo"`
	Origen       Origen                    `json:"origen"`
	Destino      Destino                   `json:"destino"`
	Resumen      string                    `json:"resumen"`
	Estado       models.EstadoCanalizacion `json:"estado"`
	Motivo       *string                   `json:"motivo"` // del último rechazo o cancelación
	CreatedAt    time.Time                 `json:"createdAt"`
	VersionTag   string                    `json:"versionTag"`
}

// Page es la página de la bandeja.
type Page struct {
	Items      []View  `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

// Acciones de §8.3.
const (
	AccionAceptar   = "aceptar"   // destino: pendiente -> aceptada (crea la asignación del área destino)
	AccionRechazar  = "rechazar"  // destino: pendiente -> rechazada (motivo obligatorio)
	AccionCompletar = "completar" // destino: aceptada -> completada; externa: origen, pendiente -> completada
	AccionCancelar  = "cancelar"  // origen: pendiente -> cancelada (motivo obligatorio)
)

// TransitionRequest cambia el estado de una canalización.
type TransitionRequest struct {
	Accion string  `json:"accion" enums:"aceptar,rechazar,completar,cancelar"`
	Motivo *string `json:"motivo,omitempty"` // 10 a 500; obligatorio en rechazar y cancelar
}

// ListQuery son los filtros de la bandeja.
type ListQuery struct {
	Direction string
	Estados   []models.EstadoCanalizacion
	RecordID  *uuid.UUID
}
