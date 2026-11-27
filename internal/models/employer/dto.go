package employer

import (
	"time"

	"backend-hackathon/internal/models"
	"github.com/google/uuid"
)

type RegistrationRequest struct {
	Nombre      string `json:"nombre"`
	Correo      string `json:"correo"`
	Telefono    string `json:"telefono"`
	Descripcion string `json:"descripcion"`
}
type RegistrationResponse struct {
	ID     uuid.UUID                      `json:"id"`
	Estado models.EstadoRegistroEmpleador `json:"estado"`
}
type LoginRequest struct {
	Correo string `json:"correo"`
}
type ChallengeResponse struct {
	ChallengeID     uuid.UUID `json:"challengeId"`
	TelefonoMascara string    `json:"telefonoMascara"`
}
type VerifyRequest struct {
	ChallengeID uuid.UUID `json:"challengeId"`
	Code        string    `json:"code"`
}
type VerifyResponse struct {
	EmployerID uuid.UUID `json:"employerId"`
}
type EmployerMe struct {
	ID             uuid.UUID                      `json:"id"`
	Nombre         string                         `json:"nombre"`
	Correo         string                         `json:"correo"`
	Telefono       string                         `json:"telefono"`
	Descripcion    string                         `json:"descripcion"`
	Estado         models.EstadoRegistroEmpleador `json:"estado"`
	MotivoRevision *string                        `json:"motivoRevision,omitempty"`
}
type UpdateRegistrationRequest struct {
	Nombre      string `json:"nombre"`
	Telefono    string `json:"telefono"`
	Descripcion string `json:"descripcion"`
}
type JobRequest struct {
	Puesto      string `json:"puesto"`
	Sector      string `json:"sector"`
	Horario     string `json:"horario"`
	TipoEmpleo  string `json:"tipoEmpleo"`
	Ubicacion   string `json:"ubicacion"`
	Descripcion string `json:"descripcion"`
	Requisitos  string `json:"requisitos"`
}
type JobView struct {
	ID            uuid.UUID            `json:"id"`
	Puesto        string               `json:"puesto"`
	Empresa       string               `json:"empresa"`
	Sector        string               `json:"sector"`
	Horario       string               `json:"horario"`
	TipoEmpleo    string               `json:"tipoEmpleo"`
	Ubicacion     string               `json:"ubicacion"`
	Descripcion   string               `json:"descripcion"`
	Requisitos    string               `json:"requisitos"`
	Estado        models.EstadoVacante `json:"estado"`
	PublicadaEn   time.Time            `json:"publicadaEn"`
	Actualizada   time.Time            `json:"actualizada"`
	Postulaciones int                  `json:"postulaciones"`
}
type ApplicationView struct {
	ID               uuid.UUID                `json:"id"`
	VacanteID        uuid.UUID                `json:"vacanteId"`
	Estado           models.EstadoPostulacion `json:"estado"`
	CreadoEn         time.Time                `json:"creadoEn"`
	ContactoLiberado bool                     `json:"contactoLiberado"`
	Nombre           *string                  `json:"nombre,omitempty"`
	Contacto         *string                  `json:"contacto,omitempty"`
}
type InterviewRequest struct {
	Fecha string `json:"fecha"`
	Lugar string `json:"lugar"`
	Notas string `json:"notas"`
}
type DecisionRequest struct {
	Decision string `json:"decision"`
	Motivo   string `json:"motivo"`
}
