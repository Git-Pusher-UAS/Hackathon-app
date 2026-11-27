package auditlog

import (
	"time"

	"github.com/google/uuid"
)

type Actor struct {
	ID            uuid.UUID `json:"id"`
	NombreLaboral string    `json:"nombreLaboral"`
}

type Entry struct {
	ID         uuid.UUID  `json:"id"`
	Fecha      time.Time  `json:"fecha"`
	Actor      *Actor     `json:"actor"`
	Rol        *string    `json:"rol"`
	Accion     string     `json:"accion"`
	Expediente *uuid.UUID `json:"expediente"`
	Detalle    *string    `json:"detalle"`
}
