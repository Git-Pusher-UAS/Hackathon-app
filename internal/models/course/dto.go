// Package course define los DTOs de /courses (§11).
package course

import (
	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

// Curso es un curso, charla o conferencia.
type Curso struct {
	ID          uuid.UUID        `json:"id"`
	Tipo        models.TipoCurso `json:"tipo" enums:"curso,charla,conferencia"`
	Titulo      string           `json:"titulo" example:"Finanzas para tu negocio"`
	Descripcion string           `json:"descripcion"`
	Fecha       string           `json:"fecha" example:"2026-12-12"` // AAAA-MM-DD
	Hora        string           `json:"hora" example:"17:00"`       // HH:MM
	Lugar       string           `json:"lugar" example:"Sala 2, Los Mochis"`
	Cupo        int              `json:"cupo" example:"25"`
	Inscritas   int              `json:"inscritas" example:"18"`
	// Inscrita solo se llena con sesión de usuaria.
	Inscrita *bool `json:"inscrita,omitempty"`
}

// CreateRequest es el cuerpo de POST /courses.
type CreateRequest struct {
	Tipo        models.TipoCurso `json:"tipo" enums:"curso,charla,conferencia"`
	Titulo      string           `json:"titulo"`
	Descripcion string           `json:"descripcion"`
	Fecha       string           `json:"fecha" example:"2026-12-12"`
	Hora        string           `json:"hora" example:"17:00"`
	Lugar       string           `json:"lugar"`
	Cupo        int              `json:"cupo" example:"25"`
}

// Asistencia es una inscripción en la lista. Solo alias: nunca nombre ni teléfono.
type Asistencia struct {
	EnrollmentID uuid.UUID `json:"enrollmentId"`
	Alias        string    `json:"alias" example:"Participante 01"`
	Presente     bool      `json:"presente"`
}

// MarcaAsistencia es un elemento del cuerpo de PUT /courses/{id}/attendance.
type MarcaAsistencia struct {
	EnrollmentID uuid.UUID `json:"enrollmentId"`
	Presente     bool      `json:"presente"`
}

// Page es la página de cursos.
type Page struct {
	Items      []Curso `json:"items"`
	NextCursor *string `json:"nextCursor"`
}
