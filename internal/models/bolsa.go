package models

import (
	"time"

	"github.com/google/uuid"
)

type Vacante struct {
	ID             uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EmpleadorID    *uuid.UUID    `gorm:"type:uuid;index"`
	Puesto         string        `gorm:"type:varchar(150);not null"`
	Empresa        string        `gorm:"type:varchar(150);not null"`
	Sector         string        `gorm:"type:varchar(100);not null"`
	Horario        string        `gorm:"type:varchar(100);not null"`
	TipoEmpleo     string        `gorm:"type:varchar(50);not null"`
	Ubicacion      string        `gorm:"type:varchar(150);not null"`
	Descripcion    string        `gorm:"type:text;not null"`
	Requisitos     string        `gorm:"type:text;not null"`
	Estado         EstadoVacante `gorm:"type:varchar(30);not null;default:'published'"`
	MotivoRevision *string       `gorm:"type:text"`
	PublicadaEn    time.Time     `gorm:"not null"`
	ActualizadaEn  time.Time     `gorm:"not null"`
	CreadoEn       time.Time     `gorm:"not null;default:now();<-:create"`
}

func (Vacante) TableName() string { return "bolsa.vacantes" }

type Postulacion struct {
	ID                  uuid.UUID         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VacanteID           uuid.UUID         `gorm:"type:uuid;not null;uniqueIndex:ux_postulacion"`
	UsuariaID           uuid.UUID         `gorm:"type:uuid;not null;uniqueIndex:ux_postulacion"`
	Estado              EstadoPostulacion `gorm:"type:varchar(30);not null;default:'submitted'"`
	EntrevistaEn        *time.Time        `gorm:"type:timestamptz"`
	EntrevistaLugar     *string           `gorm:"type:text"`
	EntrevistaNotas     *string           `gorm:"type:text"`
	DecisionMotivo      *string           `gorm:"type:text"`
	ContactoLiberado    bool              `gorm:"not null;default:false"`
	ContactoLiberadoEn  *time.Time        `gorm:"type:timestamptz"`
	ContactoLiberadoPor *uuid.UUID        `gorm:"type:uuid"`
	CreadoEn            time.Time         `gorm:"not null;default:now();<-:create"`
	ActualizadoEn       time.Time         `gorm:"not null;default:now()"`
}

func (Postulacion) TableName() string { return "bolsa.postulaciones" }
