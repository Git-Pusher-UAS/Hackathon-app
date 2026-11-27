package models

import (
	"time"

	"github.com/google/uuid"
)

type EstadoRegistroEmpleador string
type EstadoVacante string
type EstadoPostulacion string

const (
	EmpleadorPendiente EstadoRegistroEmpleador = "pending_review"
	EmpleadorAprobado  EstadoRegistroEmpleador = "approved"
	EmpleadorRechazado EstadoRegistroEmpleador = "rejected"

	VacantePendiente EstadoVacante = "pending_review"
	VacanteAprobada  EstadoVacante = "approved"
	VacanteRechazada EstadoVacante = "rejected"
	VacantePublicada EstadoVacante = "published"
	VacanteCerrada   EstadoVacante = "closed"

	PostulacionEnviada             EstadoPostulacion = "submitted"
	PostulacionEntrevistaInvitada  EstadoPostulacion = "interview_invited"
	PostulacionEntrevistaAceptada  EstadoPostulacion = "interview_accepted"
	PostulacionEntrevistaRechazada EstadoPostulacion = "interview_declined"
	PostulacionRetirada            EstadoPostulacion = "withdrawn"
	PostulacionAceptada            EstadoPostulacion = "accepted"
	PostulacionRechazada           EstadoPostulacion = "rejected"
	PostulacionContratada          EstadoPostulacion = "hired"
)

type Empleador struct {
	ID             uuid.UUID               `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Nombre         string                  `gorm:"type:varchar(150);not null"`
	Correo         string                  `gorm:"type:varchar(150);not null;uniqueIndex"`
	Telefono       string                  `gorm:"type:varchar(30);not null"`
	Descripcion    string                  `gorm:"type:text;not null;default:''"`
	Estado         EstadoRegistroEmpleador `gorm:"type:varchar(30);not null;default:'pending_review'"`
	MotivoRevision *string                 `gorm:"type:text"`
	CreadoEn       time.Time               `gorm:"not null;default:now();<-:create"`
	ActualizadoEn  time.Time               `gorm:"not null;default:now()"`
}

func (Empleador) TableName() string { return "bolsa.empleadores" }

type EmpleadorDesafioOTP struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EmpleadorID uuid.UUID  `gorm:"type:uuid;not null;index"`
	CodigoHash  []byte     `gorm:"type:bytea;not null"`
	VenceEn     time.Time  `gorm:"not null"`
	Intentos    int        `gorm:"not null;default:0"`
	ConsumidoEn *time.Time `gorm:"type:timestamptz"`
	CreadoEn    time.Time  `gorm:"not null;default:now();<-:create"`
}

func (EmpleadorDesafioOTP) TableName() string { return "bolsa.empleador_desafios_otp" }

type EmpleadorSesion struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EmpleadorID     uuid.UUID  `gorm:"type:uuid;not null;index"`
	TokenHash       []byte     `gorm:"type:bytea;not null;uniqueIndex"`
	CSRFHash        []byte     `gorm:"type:bytea;not null"`
	UltimaActividad time.Time  `gorm:"not null"`
	VenceAbsoluto   time.Time  `gorm:"not null"`
	RevocadoEn      *time.Time `gorm:"type:timestamptz"`
	CreadoEn        time.Time  `gorm:"not null;default:now();<-:create"`
}

func (EmpleadorSesion) TableName() string { return "bolsa.empleador_sesiones" }

type CVPerfil struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CuentaID      uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	Nombre        string    `gorm:"type:varchar(150);not null"`
	Titular       string    `gorm:"type:varchar(150);not null;default:''"`
	Resumen       string    `gorm:"type:text;not null;default:''"`
	Contacto      string    `gorm:"type:varchar(200);not null;default:''"`
	Habilidades   string    `gorm:"type:text;not null;default:''"` // JSON array string
	ActualizadoEn time.Time `gorm:"not null;default:now()"`
	CreadoEn      time.Time `gorm:"not null;default:now();<-:create"`
}

func (CVPerfil) TableName() string { return "portal.cv_perfiles" }

type CVExperiencia struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CuentaID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Empresa     string    `gorm:"type:varchar(150);not null"`
	Puesto      string    `gorm:"type:varchar(150);not null"`
	Inicio      string    `gorm:"type:varchar(30);not null;default:''"`
	Fin         string    `gorm:"type:varchar(30);not null;default:''"`
	Descripcion string    `gorm:"type:text;not null;default:''"`
}

func (CVExperiencia) TableName() string { return "portal.cv_experiencias" }

type CVEstudio struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CuentaID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Institucion string    `gorm:"type:varchar(150);not null"`
	Grado       string    `gorm:"type:varchar(150);not null"`
	Inicio      string    `gorm:"type:varchar(30);not null;default:''"`
	Fin         string    `gorm:"type:varchar(30);not null;default:''"`
}

func (CVEstudio) TableName() string { return "portal.cv_estudios" }
