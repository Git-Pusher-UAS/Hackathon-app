// Package institutionalcontact define la respuesta de GET /api/institutional-contacts.
package institutionalcontact

import "github.com/google/uuid"

type Contact struct {
	ID          uuid.UUID `json:"id"`
	Nombre      string    `json:"nombre"`
	Descripcion string    `json:"descripcion"`
	Telefono    string    `json:"telefono"`
	Horario     string    `json:"horario"`
	Prioritario bool      `json:"prioritario"`
}
