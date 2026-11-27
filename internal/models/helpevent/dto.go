// Package helpevent define el cuerpo de POST /help-events (API-040).
package helpevent

import "time"

// CreateRequest registra que el personal abrió el marcador hacia un contacto
// de ayuda. La idempotencia va en el header X-Operation-Id.
type CreateRequest struct {
	Action     string    `json:"action" example:"dialer_opened"`
	OccurredAt time.Time `json:"occurredAt" example:"2026-11-25T16:42:00-07:00"`
}
