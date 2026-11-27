// Package helpevent define el cuerpo de POST /api/help-events.
package helpevent

import (
	"encoding/json"
	"time"
)

type CreateRequest struct {
	ContactID   *string   `json:"contactId,omitempty"`
	Action      string    `json:"action"`
	OccurredAt  time.Time `json:"occurredAt"`
	OperationID string    `json:"operationId"`
}

func (r *CreateRequest) UnmarshalJSON(data []byte) error {
	type alias CreateRequest
	var aux struct {
		alias
		Accion     string    `json:"accion"`
		OcurridoEn time.Time `json:"ocurridoEn"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*r = CreateRequest(aux.alias)
	if r.Action == "" {
		r.Action = aux.Accion
	}
	if r.OccurredAt.IsZero() {
		r.OccurredAt = aux.OcurridoEn
	}
	return nil
}
