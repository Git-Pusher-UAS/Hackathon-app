package course

import (
	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

type Curso struct {
	ID          uuid.UUID        `json:"id"`
	Tipo        models.TipoCurso `json:"tipo"`
	Titulo      string           `json:"titulo"`
	Descripcion string           `json:"descripcion"`
	Fecha       string           `json:"fecha"`
	Hora        string           `json:"hora"`
	Lugar       string           `json:"lugar"`
	Cupo        int              `json:"cupo"`
	Inscritas   int              `json:"inscritas"`
	Inscrita    *bool            `json:"inscrita,omitempty"`
}

type CreateRequest struct {
	Tipo        models.TipoCurso `json:"tipo"`
	Titulo      string           `json:"titulo"`
	Descripcion string           `json:"descripcion"`
	Fecha       string           `json:"fecha"`
	Hora        string           `json:"hora"`
	Lugar       string           `json:"lugar"`
	Cupo        int              `json:"cupo"`
}

type Asistencia struct {
	ID       uuid.UUID `json:"id"`
	Alias    string    `json:"alias"`
	Presente bool      `json:"presente"`
}

type MarcaAsistencia struct {
	EnrollmentID uuid.UUID `json:"enrollmentId"`
	Presente     bool      `json:"presente"`
}
