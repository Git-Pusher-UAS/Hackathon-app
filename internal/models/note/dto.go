// Package note define los DTOs de POST /notes/sync (API-041).
package note

import (
	"time"

	"github.com/google/uuid"
)

// MaxBatch es el máximo de notas por lote.
const MaxBatch = 20

// Resultados por nota.
const (
	ResultApplied   = "applied"   // guardada (o ya estaba guardada con ese operationId)
	ResultBlocked   = "blocked"   // no se guardará nunca así: corregir o descartar el borrador
	ResultConflict  = "conflict"  // choca con el estado actual
	ResultRetryable = "retryable" // falla temporal: reintentar con el mismo operationId
)

// SyncNote es un borrador guardado sin conexión. operationId hace idempotente
// cada nota por separado.
type SyncNote struct {
	OperationID   string    `json:"operationId" example:"7d1f4c1e-5d9a-4f4f-8a43-2f3b4c5d6e7f"`
	RecordID      string    `json:"recordId"`
	DepartmentID  string    `json:"departmentId"`
	Texto         string    `json:"texto"`
	OcurridoEn    time.Time `json:"ocurridoEn"`
	CorrigeNotaID *string   `json:"corrigeNotaId,omitempty"`
}

// SyncRequest es el lote (1 a 20 notas).
type SyncRequest struct {
	Notes []SyncNote `json:"notes"`
}

// SyncResult es el resultado de una nota, en el mismo orden del lote.
type SyncResult struct {
	OperationID string     `json:"operationId"`
	Result      string     `json:"result" enums:"applied,blocked,conflict,retryable"`
	NoteID      *uuid.UUID `json:"noteId,omitempty"`
	Code        string     `json:"code,omitempty" example:"CASE_CLOSED"`
}

// SyncResponse lista los resultados.
type SyncResponse struct {
	Results []SyncResult `json:"results"`
}
