package note

import (
	"time"

	"github.com/google/uuid"
)

const MaxBatch = 20

const (
	ResultApplied   = "applied"
	ResultBlocked   = "blocked"
	ResultConflict  = "conflict"
	ResultRetryable = "retryable"
)

type SyncNote struct {
	OperationID    string    `json:"operationId"`
	RecordID       string    `json:"recordId"`
	DepartmentID   string    `json:"departmentId"`
	Texto          string    `json:"texto"`
	OcurridoEn     time.Time `json:"ocurridoEn"`
	CorrigeNotaID  *string   `json:"corrigeNotaId,omitempty"`
	BaseVersionTag *string   `json:"baseVersionTag,omitempty"`
}

type SyncRequest struct {
	Notes []SyncNote `json:"notes"`
}

type SyncResult struct {
	ID     string     `json:"id"`
	Result string     `json:"result"`
	NoteID *uuid.UUID `json:"noteId,omitempty"`
	Reason string     `json:"reason,omitempty"`
}
