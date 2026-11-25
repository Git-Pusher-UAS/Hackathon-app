package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// A01 · catalogos.departamentos
type Departamento struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Codigo      string    `gorm:"type:varchar(40);not null;uniqueIndex"`
	Nombre      string    `gorm:"type:varchar(100);not null"`
	Asistencial bool      `gorm:"not null"`
	Activo      bool      `gorm:"not null;default:true"`
	CreadoEn    time.Time `gorm:"not null;default:now();<-:create"`
}

func (Departamento) TableName() string { return "catalogos.departamentos" }

// A02 · catalogos.roles
type Rol struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Codigo   string    `gorm:"type:varchar(50);not null;uniqueIndex"`
	Nombre   string    `gorm:"type:varchar(100);not null"`
	Activo   bool      `gorm:"not null;default:true"`
	CreadoEn time.Time `gorm:"not null;default:now();<-:create"`
}

func (Rol) TableName() string { return "catalogos.roles" }

// A03 · catalogos.permisos
type Permiso struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Codigo      string    `gorm:"type:varchar(100);not null;uniqueIndex"`
	Descripcion string    `gorm:"type:varchar(250);not null"`
	CreadoEn    time.Time `gorm:"not null;default:now();<-:create"`
}

func (Permiso) TableName() string { return "catalogos.permisos" }

// A04 · catalogos.rol_permisos (UNIQUE rol_id, permiso_id)
type RolPermiso struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RolID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_rol_permiso"`
	PermisoID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_rol_permiso"`
	CreadoEn  time.Time `gorm:"not null;default:now();<-:create"`
}

func (RolPermiso) TableName() string { return "catalogos.rol_permisos" }

// A05 · catalogos.sectores. Nunca puntos domiciliarios.
type Sector struct {
	ID                 uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Codigo             string          `gorm:"type:varchar(40);not null;uniqueIndex"`
	Nombre             string          `gorm:"type:varchar(150);not null"`
	Tipo               TipoSector      `gorm:"type:varchar(20);not null"`
	Geometria          json.RawMessage `gorm:"type:jsonb"` // GeoJSON institucional; NULL permitido
	VersionCartografia string          `gorm:"type:varchar(50);not null"`
	Activo             bool            `gorm:"not null;default:true"`
	CreadoEn           time.Time       `gorm:"not null;default:now();<-:create"`
}

func (Sector) TableName() string { return "catalogos.sectores" }

// A06 · catalogos.tipos_caso
type TipoCaso struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Codigo   string    `gorm:"type:varchar(50);not null;uniqueIndex"`
	Nombre   string    `gorm:"type:varchar(150);not null"`
	Activo   bool      `gorm:"not null;default:true"`
	CreadoEn time.Time `gorm:"not null;default:now();<-:create"`
}

func (TipoCaso) TableName() string { return "catalogos.tipos_caso" }

// A07 · catalogos.instituciones (con control de versión / ETag)
type Institucion struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Nombre        string     `gorm:"type:varchar(150);not null"`
	Telefono      string     `gorm:"type:varchar(20);not null"`
	Horario       string     `gorm:"type:varchar(250);not null"`
	RutaAtencion  string     `gorm:"type:varchar(1000);not null"`
	Publicable    bool       `gorm:"not null"`
	VerificadoEn  *time.Time `gorm:"type:timestamptz"`
	Activo        bool       `gorm:"not null;default:true"`
	Version       int64      `gorm:"not null;default:1"`
	ActualizadoEn time.Time  `gorm:"not null;default:now()"`
	CreadoEn      time.Time  `gorm:"not null;default:now();<-:create"`
}

func (Institucion) TableName() string { return "catalogos.instituciones" }

// A08 · catalogos.propositos (finalidades del consentimiento)
type Proposito struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Codigo      string    `gorm:"type:varchar(80);not null;uniqueIndex"`
	Descripcion string    `gorm:"type:varchar(500);not null"`
	Opcional    bool      `gorm:"not null"`
	Activo      bool      `gorm:"not null;default:true"`
	Revocable   bool      `gorm:"not null"`
	CreadoEn    time.Time `gorm:"not null;default:now();<-:create"`
}

func (Proposito) TableName() string { return "catalogos.propositos" }

// A09 · catalogos.avisos_privacidad (versiones inmutables)
type AvisoPrivacidad struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CodigoVersion string     `gorm:"type:varchar(50);not null;uniqueIndex"`
	Texto         string     `gorm:"type:text;not null"`
	PublicadoEn   time.Time  `gorm:"not null"`
	RetiradoEn    *time.Time `gorm:"type:timestamptz"`
	CreadoEn      time.Time  `gorm:"not null;default:now();<-:create"`
}

func (AvisoPrivacidad) TableName() string { return "catalogos.avisos_privacidad" }

// A10 · catalogos.aviso_propositos (UNIQUE aviso_id, proposito_id)
type AvisoProposito struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AvisoID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_aviso_proposito"`
	PropositoID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_aviso_proposito"`
	CreadoEn    time.Time `gorm:"not null;default:now();<-:create"`
}

func (AvisoProposito) TableName() string { return "catalogos.aviso_propositos" }
