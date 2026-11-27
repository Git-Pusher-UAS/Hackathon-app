package models

import (
	"time"

	"github.com/google/uuid"
)

// TipoCurso es el tipo de evento de Capacitación.
type TipoCurso string

const (
	CursoCurso       TipoCurso = "curso"
	CursoCharla      TipoCurso = "charla"
	CursoConferencia TipoCurso = "conferencia"
)

// Curso es un curso, charla o conferencia (capacitacion.cursos). Se publica
// en el foro de la usuaria al crearse.
type Curso struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Tipo        TipoCurso `gorm:"type:varchar(15);not null"`
	Titulo      string    `gorm:"type:varchar(150);not null"`
	Descripcion string    `gorm:"type:varchar(2000);not null"`
	Fecha       time.Time `gorm:"type:date;not null;index"`
	Hora        string    `gorm:"type:varchar(5);not null"` // HH:MM, 24 h
	Lugar       string    `gorm:"type:varchar(250);not null"`
	Cupo        int       `gorm:"not null"`
	CreadoPor   uuid.UUID `gorm:"type:uuid;not null"`
	CreadoEn    time.Time `gorm:"not null;default:now();<-:create"`
}

func (Curso) TableName() string { return "capacitacion.cursos" }

// Inscripcion es la inscripción de una usuaria a un curso. Para la lista de
// asistencia solo se usa el alias, nunca nombre ni teléfono.
type Inscripcion struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CursoID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_inscripcion"`
	UsuariaID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_inscripcion"`
	Alias     string    `gorm:"type:varchar(60);not null"`
	Presente  bool      `gorm:"not null;default:false"`
	CreadoEn  time.Time `gorm:"not null;default:now();<-:create"`
}

func (Inscripcion) TableName() string { return "capacitacion.inscripciones" }
