// Package auditlog define la respuesta de GET /audit-logs (§13.1).
package auditlog

import (
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

// Actor es quien hizo la acción.
type Actor struct {
	ID            uuid.UUID `json:"id"`
	NombreLaboral string    `json:"nombreLaboral"`
}

// Entry es un evento de la bitácora. No incluye la IP de origen.
type Entry struct {
	ID           uuid.UUID                 `json:"id"`
	Fecha        time.Time                 `json:"fecha"`
	Actor        *Actor                    `json:"actor"` // null en acciones anónimas
	Rol          *string                   `json:"rol"`
	Accion       string                    `json:"accion" example:"casefile.disclose.telefono"`
	ExpedienteID *uuid.UUID                `json:"expedienteId"`
	Resultado    models.ResultadoAuditoria `json:"resultado"`
	Detalle      *string                   `json:"detalle" example:"Reveló teléfono. Motivo: Contacto de seguimiento"`
	RequestID    uuid.UUID                 `json:"requestId"`
}

// Page es la página de la bitácora.
type Page struct {
	Items      []Entry `json:"items"`
	NextCursor *string `json:"nextCursor"`
}
