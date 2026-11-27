package models

import (
	"time"

	"github.com/google/uuid"
)

type PortalCuenta struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Alias              string    `gorm:"type:varchar(80);not null;uniqueIndex"`
	Nombre             string    `gorm:"type:varchar(150);not null"`
	CURPNormalizada    string    `gorm:"column:curp_normalizada;type:varchar(18);not null;uniqueIndex"`
	Telefono           string    `gorm:"type:varchar(30);not null"`
	AceptaAviso        bool      `gorm:"not null;default:false"`
	SilenciarAvisos    bool      `gorm:"not null;default:false"`
	CompartirUbicacion bool      `gorm:"not null;default:false"`
	NombreDiscreto     bool      `gorm:"not null;default:false"`
	CreadoEn           time.Time `gorm:"not null;default:now();<-:create"`
	ActualizadoEn      time.Time `gorm:"not null;default:now()"`
}

func (PortalCuenta) TableName() string { return "portal.cuentas" }

type PortalContactoConfianza struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CuentaID uuid.UUID `gorm:"type:uuid;not null;index"`
	Nombre   string    `gorm:"type:varchar(150);not null"`
	Telefono string    `gorm:"type:varchar(30);not null"`
	CreadoEn time.Time `gorm:"not null;default:now();<-:create"`
}

func (PortalContactoConfianza) TableName() string { return "portal.contactos_confianza" }

type PortalDesafioOTP struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CuentaID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	CodigoHash  []byte     `gorm:"type:bytea;not null"`
	VenceEn     time.Time  `gorm:"not null"`
	Intentos    int        `gorm:"not null;default:0"`
	ConsumidoEn *time.Time `gorm:"type:timestamptz"`
	CreadoEn    time.Time  `gorm:"not null;default:now();<-:create"`
}

func (PortalDesafioOTP) TableName() string { return "portal.desafios_otp" }

type PortalSesion struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CuentaID        uuid.UUID  `gorm:"type:uuid;not null;index"`
	TokenHash       []byte     `gorm:"type:bytea;not null;uniqueIndex"`
	CSRFHash        []byte     `gorm:"type:bytea;not null"`
	UltimaActividad time.Time  `gorm:"not null"`
	VenceAbsoluto   time.Time  `gorm:"not null"`
	RevocadoEn      *time.Time `gorm:"type:timestamptz"`
	CreadoEn        time.Time  `gorm:"not null;default:now();<-:create"`
}

func (PortalSesion) TableName() string { return "portal.sesiones" }

type EstadoAlertaAyuda string

const (
	AlertaAyudaActiva    EstadoAlertaAyuda = "activa"
	AlertaAyudaCancelada EstadoAlertaAyuda = "cancelada"
)

type PortalAlertaAyuda struct {
	ID                  uuid.UUID         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CuentaID            uuid.UUID         `gorm:"type:uuid;not null;index"`
	OperacionID         uuid.UUID         `gorm:"type:uuid;not null;uniqueIndex"`
	Estado              EstadoAlertaAyuda `gorm:"type:varchar(20);not null"`
	Lat                 *float64          `gorm:"type:double precision"`
	Lon                 *float64          `gorm:"type:double precision"`
	UbicacionCompartida bool              `gorm:"not null;default:false"`
	ContactosAvisados   bool              `gorm:"not null;default:false"`
	MotivoCancelacion   *string           `gorm:"type:varchar(500)"`
	CreadoEn            time.Time         `gorm:"not null;default:now();<-:create"`
	ActualizadoEn       time.Time         `gorm:"not null;default:now()"`
}

func (PortalAlertaAyuda) TableName() string { return "portal.alertas_ayuda" }
