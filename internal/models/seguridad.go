package models

import (
	"time"

	"github.com/google/uuid"
)

// A19 · seguridad.claves_registro. UNIQUE(modulo, tipo_registro, registro_id).
// Nunca contiene la KEK en claro.
type ClaveRegistro struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Modulo         string     `gorm:"type:varchar(40);not null;uniqueIndex:ux_clave_registro"`
	TipoRegistro   string     `gorm:"type:varchar(60);not null;uniqueIndex:ux_clave_registro"`
	RegistroID     uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:ux_clave_registro"`
	DEKEnvuelta    []byte     `gorm:"column:dek_envuelta;type:bytea;not null"`
	VersionKEK     int        `gorm:"column:version_kek;not null"`
	NonceEnvoltura []byte     `gorm:"type:bytea;not null"`
	Algoritmo      string     `gorm:"type:varchar(30);not null"` // AES-256-GCM
	RotadaEn       *time.Time `gorm:"type:timestamptz"`
	CreadoEn       time.Time  `gorm:"not null;default:now();<-:create"`
}

func (ClaveRegistro) TableName() string { return "seguridad.claves_registro" }

// A20 · seguridad.operaciones (idempotencia).
// UNIQUE(actor_id, tipo_operacion, operation_id).
type Operacion struct {
	ID               uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ActorID          uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex:ux_operacion"`
	TipoOperacion    string          `gorm:"type:varchar(100);not null;uniqueIndex:ux_operacion"`
	OperationID      uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex:ux_operacion"` // lo genera el cliente
	RecursoAmbito    *uuid.UUID      `gorm:"type:uuid"`
	PeticionHMAC     []byte          `gorm:"column:peticion_hmac;type:bytea;not null"`
	Estado           EstadoOperacion `gorm:"type:varchar(15);not null"`
	HTTPStatus       int16           `gorm:"column:http_status;not null"`
	ResultadoCifrado []byte          `gorm:"type:bytea"` // se anula al vencer, la marca permanece
	ResultadoVenceEn time.Time       `gorm:"not null"`
	AplicadaEn       time.Time       `gorm:"not null"`
	ClaveID          uuid.UUID       `gorm:"type:uuid;not null"`
	CreadoEn         time.Time       `gorm:"not null;default:now();<-:create"`
}

func (Operacion) TableName() string { return "seguridad.operaciones" }
