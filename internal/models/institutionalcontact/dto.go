// Package institutionalcontact define los DTOs de GET /institutional-contacts (API-039).
package institutionalcontact

import (
	"time"

	"github.com/google/uuid"
)

// Contact es una institución publicable del directorio de ayuda.
type Contact struct {
	ID           uuid.UUID  `json:"id"`
	Nombre       string     `json:"nombre"`
	Telefono     string     `json:"telefono" example:"911"`
	Horario      string     `json:"horario" example:"24 horas"`
	RutaAtencion string     `json:"rutaAtencion"`
	VerificadoEn *time.Time `json:"verificadoEn"`
}

// Response es el directorio completo. Version cambia cuando cambia algún contacto.
type Response struct {
	Items   []Contact `json:"items"`
	Version string    `json:"version"`
}
