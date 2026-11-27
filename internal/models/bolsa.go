package models

import (
	"time"

	"github.com/google/uuid"
)

type Vacante struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Puesto        string    `gorm:"type:varchar(150);not null"`
	Empresa       string    `gorm:"type:varchar(150);not null"`
	Sector        string    `gorm:"type:varchar(100);not null"`
	Horario       string    `gorm:"type:varchar(100);not null"`
	TipoEmpleo    string    `gorm:"type:varchar(50);not null"`
	Ubicacion     string    `gorm:"type:varchar(150);not null"`
	Descripcion   string    `gorm:"type:text;not null"`
	Requisitos    string    `gorm:"type:text;not null"`
	PublicadaEn   time.Time `gorm:"not null"`
	ActualizadaEn time.Time `gorm:"not null"`
	CreadoEn      time.Time `gorm:"not null;default:now();<-:create"`
}

func (Vacante) TableName() string { return "bolsa.vacantes" }

type Postulacion struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VacanteID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_postulacion"`
	UsuariaID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_postulacion"`
	CreadoEn  time.Time `gorm:"not null;default:now();<-:create"`
}

func (Postulacion) TableName() string { return "bolsa.postulaciones" }
