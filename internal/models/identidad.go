package models

import (
	"time"

	"github.com/google/uuid"
)

// A11 · identidad.actores. Atribución común de personal y procesos del sistema.
type Actor struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Tipo     TipoActor `gorm:"type:varchar(15);not null"`
	Activo   bool      `gorm:"not null;default:true"`
	CreadoEn time.Time `gorm:"not null;default:now();<-:create"`
}

func (Actor) TableName() string { return "identidad.actores" }

// A12 · identidad.personal. El ID es a la vez PK y FK a identidad.actores.id
// (sin default: se toma del actor recién creado).
type Personal struct {
	ID                     uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Usuario                string         `gorm:"type:varchar(100);not null"` // UNIQUE sobre lower(usuario)
	NombreLaboral          string         `gorm:"type:varchar(150);not null"`
	RolID                  uuid.UUID      `gorm:"type:uuid;not null"`
	DepartamentoID         uuid.UUID      `gorm:"type:uuid;not null"`
	PasswordHash           string         `gorm:"type:text;not null"` // Argon2id (PHC)
	RequiereCambioPassword bool           `gorm:"not null;default:true"`
	Estado                 EstadoPersonal `gorm:"type:varchar(15);not null"`
	VersionPermisos        int64          `gorm:"not null;default:1"`
	BloqueadoHasta         *time.Time     `gorm:"type:timestamptz"`
	FallosConsecutivos     int            `gorm:"not null;default:0"`
	Version                int64          `gorm:"not null;default:1"`
	ActualizadoEn          time.Time      `gorm:"not null;default:now()"`
	CreadoEn               time.Time      `gorm:"not null;default:now();<-:create"`
}

func (Personal) TableName() string { return "identidad.personal" }

// A13 · identidad.delegaciones
type Delegacion struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PersonalID    uuid.UUID  `gorm:"type:uuid;not null"`
	PermisoID     uuid.UUID  `gorm:"type:uuid;not null"`
	OtorgadoPor   uuid.UUID  `gorm:"type:uuid;not null"`
	Inicio        time.Time  `gorm:"not null"`
	Fin           time.Time  `gorm:"not null"` // CHECK fin > inicio
	RevocadoEn    *time.Time `gorm:"type:timestamptz"`
	MotivoCifrado []byte     `gorm:"type:bytea;not null"`
	ClaveID       uuid.UUID  `gorm:"type:uuid;not null"`
	CreadoEn      time.Time  `gorm:"not null;default:now();<-:create"`
}

func (Delegacion) TableName() string { return "identidad.delegaciones" }

// A14 · identidad.mfa_personal (UNIQUE personal_id)
type MFAPersonal struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PersonalID      uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	SemillaCifrada  []byte     `gorm:"type:bytea;not null"`
	Estado          EstadoMFA  `gorm:"type:varchar(15);not null"`
	UltimoPasoUsado *int64     `gorm:"type:bigint"`
	ConfirmadoEn    *time.Time `gorm:"type:timestamptz"`
	ClaveID         uuid.UUID  `gorm:"type:uuid;not null"`
	Version         int64      `gorm:"not null;default:1"`
	ActualizadoEn   time.Time  `gorm:"not null;default:now()"`
	CreadoEn        time.Time  `gorm:"not null;default:now();<-:create"`
}

func (MFAPersonal) TableName() string { return "identidad.mfa_personal" }

// A15 · identidad.sesiones_personal
type SesionPersonal struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TitularID       uuid.UUID  `gorm:"type:uuid;not null;index:ix_sesion_titular,priority:1"`
	TokenHash       []byte     `gorm:"type:bytea;not null;uniqueIndex"`
	CSRFHash        []byte     `gorm:"column:csrf_hash;type:bytea;not null"`
	UltimaActividad time.Time  `gorm:"not null"`
	VenceAbsoluto   time.Time  `gorm:"not null"`
	RevocadoEn      *time.Time `gorm:"type:timestamptz;index:ix_sesion_titular,priority:2"`
	Dispositivo     string     `gorm:"type:varchar(100);not null"`
	MFAVerificadoEn time.Time  `gorm:"column:mfa_verificado_en;not null"`
	VersionPermisos int64      `gorm:"not null"`
	CreadoEn        time.Time  `gorm:"not null;default:now();<-:create"`
}

func (SesionPersonal) TableName() string { return "identidad.sesiones_personal" }

// A16 · identidad.contextos_auth (bootstrap, MFA y CSRF)
type ContextoAuth struct {
	ID             uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Fase           FaseContexto `gorm:"type:varchar(15);not null"`
	PersonalID     *uuid.UUID   `gorm:"type:uuid"`
	TokenHash      []byte       `gorm:"type:bytea;not null;uniqueIndex"`
	CSRFHash       []byte       `gorm:"column:csrf_hash;type:bytea;not null"`
	EnrolamientoID *uuid.UUID   `gorm:"type:uuid"` // FK identidad.mfa_personal.id
	VenceEn        time.Time    `gorm:"not null"`
	Intentos       int          `gorm:"not null;default:0"` // CHECK 0..5
	ConsumidoEn    *time.Time   `gorm:"type:timestamptz"`
	CreadoEn       time.Time    `gorm:"not null;default:now();<-:create"`
}

func (ContextoAuth) TableName() string { return "identidad.contextos_auth" }

// A17 · identidad.control_intentos. UNIQUE(ambito, clave_hmac, ventana_inicio).
type ControlIntentos struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Ambito         string     `gorm:"type:varchar(30);not null;uniqueIndex:ux_control_intentos"`
	ClaveHMAC      []byte     `gorm:"column:clave_hmac;type:bytea;not null;uniqueIndex:ux_control_intentos"`
	VentanaInicio  time.Time  `gorm:"not null;uniqueIndex:ux_control_intentos"`
	Contador       int        `gorm:"not null;default:0"`
	BloqueadoHasta *time.Time `gorm:"type:timestamptz"`
	CreadoEn       time.Time  `gorm:"not null;default:now();<-:create"`
}

func (ControlIntentos) TableName() string { return "identidad.control_intentos" }

// A18 · identidad.recuperaciones_mfa
type RecuperacionMFA struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PersonalID      uuid.UUID  `gorm:"type:uuid;not null"`
	AdministradorID uuid.UUID  `gorm:"type:uuid;not null"`
	VerificadoEn    time.Time  `gorm:"not null"`
	MotivoCifrado   []byte     `gorm:"type:bytea;not null"`
	CompletadoEn    *time.Time `gorm:"type:timestamptz"`
	ClaveID         uuid.UUID  `gorm:"type:uuid;not null"`
	CreadoEn        time.Time  `gorm:"not null;default:now();<-:create"`
}

func (RecuperacionMFA) TableName() string { return "identidad.recuperaciones_mfa" }
