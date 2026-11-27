// Package institutionalcontact define la respuesta de GET /institutional-contacts (§9.1).
package institutionalcontact

import "github.com/google/uuid"

// Contact es una institución publicable del directorio de ayuda.
type Contact struct {
	ID          uuid.UUID `json:"id"`
	Nombre      string    `json:"nombre" example:"Emergencias"`
	Descripcion string    `json:"descripcion" example:"Si estás en peligro inmediato"`
	Telefono    string    `json:"telefono" example:"911"`
	Horario     string    `json:"horario" example:"24 horas"`
	Prioritario bool      `json:"prioritario"`
}
