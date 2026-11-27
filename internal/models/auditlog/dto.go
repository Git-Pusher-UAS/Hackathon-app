// Package auditlog define los DTOs de GET /audit-logs (API-035) y
// GET /audit-alerts (API-036).
package auditlog

import (
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

// Entry es un evento de la bitácora. No incluye la IP de origen.
type Entry struct {
	ID           uuid.UUID                 `json:"id"`
	Secuencia    int64                     `json:"secuencia"`
	ActorID      *uuid.UUID                `json:"actorId"`
	ActorUsuario *string                   `json:"actorUsuario"`
	Accion       string                    `json:"accion" example:"casefile.notes.create"`
	TipoRecurso  string                    `json:"tipoRecurso" example:"expediente"`
	RecursoID    *uuid.UUID                `json:"recursoId"`
	Resultado    models.ResultadoAuditoria `json:"resultado"`
	OcurridoEn   time.Time                 `json:"ocurridoEn"`
	RequestID    uuid.UUID                 `json:"requestId"`
}

// Alert es una alerta de acceso inusual.
type Alert struct {
	ID               uuid.UUID           `json:"id"`
	ActorObservadoID uuid.UUID           `json:"actorObservadoId"`
	Tipo             string              `json:"tipo"`
	VentanaInicio    time.Time           `json:"ventanaInicio"`
	VentanaFin       time.Time           `json:"ventanaFin"`
	Conteo           int                 `json:"conteo"`
	Umbral           int                 `json:"umbral"`
	Nivel            models.NivelAlerta  `json:"nivel"`
	Estado           models.EstadoAlerta `json:"estado"`
	CreatedAt        time.Time           `json:"createdAt"`
}

// EntryPage es la página de la bitácora.
type EntryPage struct {
	Items      []Entry `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

// AlertPage es la página de alertas.
type AlertPage struct {
	Items      []Alert `json:"items"`
	NextCursor *string `json:"nextCursor"`
}
