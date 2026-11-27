// Package helpevent define el cuerpo de POST /help-events (§9.2).
package helpevent

import "time"

// CreateRequest registra que el personal abrió el marcador hacia un contacto
// del directorio. operationId debe ser igual al encabezado Idempotency-Key.
type CreateRequest struct {
	ContactID   *string   `json:"contactId,omitempty"`
	Accion      string    `json:"accion" example:"dialer_opened"`
	OcurridoEn  time.Time `json:"ocurridoEn" example:"2026-11-26T17:42:00Z"`
	OperationID string    `json:"operationId,omitempty"`
}
