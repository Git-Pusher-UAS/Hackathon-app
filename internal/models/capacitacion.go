package models

import (
	"time"

	"github.com/google/uuid"
)

type TipoCurso string

const (
	CursoCurso       TipoCurso = "curso"
	CursoCharla      TipoCurso = "charla"
	CursoConferencia TipoCurso = "conferencia"
)

type Curso struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Tipo        TipoCurso `gorm:"type:varchar(15);not null"`
	Titulo      string    `gorm:"type:varchar(150);not null"`
	Descripcion string    `gorm:"type:varchar(2000);not null"`
	Fecha       time.Time `gorm:"type:date;not null;index"`
	Hora        string    `gorm:"type:varchar(5);not null"`
	Lugar       string    `gorm:"type:varchar(250);not null"`
	Cupo        int       `gorm:"not null"`
	CreadoPor   uuid.UUID `gorm:"type:uuid;not null"`
	CreadoEn    time.Time `gorm:"not null;default:now();<-:create"`
}

func (Curso) TableName() string { return "capacitacion.cursos" }

type Inscripcion struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CursoID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_inscripcion"`
	UsuariaID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_inscripcion"`
	Alias     string    `gorm:"type:varchar(60);not null"`
	Presente  bool      `gorm:"not null;default:false"`
	CreadoEn  time.Time `gorm:"not null;default:now();<-:create"`
}

func (Inscripcion) TableName() string { return "capacitacion.inscripciones" }

type EstadoVisita string

const (
	VisitaPlaneada  EstadoVisita = "planeada"
	VisitaRealizada EstadoVisita = "realizada"
)

type Visita struct {
	ID          uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SectorID    uuid.UUID    `gorm:"type:uuid;not null"`
	Fecha       time.Time    `gorm:"type:date;not null;index"`
	Tema        string       `gorm:"type:varchar(250);not null"`
	Responsable string       `gorm:"type:varchar(150);not null"`
	Estado      EstadoVisita `gorm:"type:varchar(15);not null"`
	CreadoPor   uuid.UUID    `gorm:"type:uuid;not null"`
	CreadoEn    time.Time    `gorm:"not null;default:now();<-:create"`
}

func (Visita) TableName() string { return "capacitacion.visitas" }
