package job

import (
	"time"

	"github.com/google/uuid"
)

type Job struct {
	ID            uuid.UUID `json:"id"`
	Puesto        string    `json:"puesto"`
	Empresa       string    `json:"empresa"`
	Sector        string    `json:"sector"`
	Horario       string    `json:"horario"`
	TipoEmpleo    string    `json:"tipoEmpleo"`
	Ubicacion     string    `json:"ubicacion"`
	Descripcion   string    `json:"descripcion"`
	Requisitos    string    `json:"requisitos"`
	PublicadaEn   time.Time `json:"publicadaEn"`
	Actualizada   time.Time `json:"actualizada"`
	Postulada     *bool     `json:"postulada,omitempty"`
	Postulaciones *int      `json:"postulaciones,omitempty"`
}

type CreateRequest struct {
	Puesto      string `json:"puesto"`
	Empresa     string `json:"empresa"`
	Sector      string `json:"sector"`
	Horario     string `json:"horario"`
	TipoEmpleo  string `json:"tipoEmpleo"`
	Ubicacion   string `json:"ubicacion"`
	Descripcion string `json:"descripcion"`
	Requisitos  string `json:"requisitos"`
}
