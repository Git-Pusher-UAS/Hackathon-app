package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type TipoPregunta string

const (
	PreguntaEscala  TipoPregunta = "escala"
	PreguntaOpcion  TipoPregunta = "opcion"
	PreguntaAbierta TipoPregunta = "abierta"
)

type Encuesta struct {
	ID        uuid.UUID          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Titulo    string             `gorm:"type:varchar(150);not null"`
	Minutos   int                `gorm:"not null"`
	Activa    bool               `gorm:"not null;default:true"`
	CreadoEn  time.Time          `gorm:"not null;default:now();<-:create"`
	Preguntas []PreguntaEncuesta `gorm:"foreignKey:EncuestaID"`
}

func (Encuesta) TableName() string { return "encuestas.encuestas" }

type PreguntaEncuesta struct {
	ID         uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EncuestaID uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex:ux_pregunta_clave"`
	Clave      string          `gorm:"type:varchar(20);not null;uniqueIndex:ux_pregunta_clave"`
	Orden      int             `gorm:"not null"`
	Texto      string          `gorm:"type:varchar(300);not null"`
	Tipo       TipoPregunta    `gorm:"type:varchar(10);not null"`
	Opciones   json.RawMessage `gorm:"type:jsonb"`
}

func (PreguntaEncuesta) TableName() string { return "encuestas.preguntas" }

type RespuestaEncuesta struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EncuestaID uuid.UUID `gorm:"type:uuid;not null;index"`
	PreguntaID uuid.UUID `gorm:"type:uuid;not null;index"`
	EnvioID    uuid.UUID `gorm:"type:uuid;not null"`
	Numero     *float64  `gorm:"type:numeric(4,2)"`
	Texto      *string   `gorm:"type:varchar(1000)"`
	CreadoEn   time.Time `gorm:"not null;default:now();<-:create"`
}

func (RespuestaEncuesta) TableName() string { return "encuestas.respuestas" }

type AsignacionEncuesta struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EncuestaID   uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:ux_asignacion"`
	UsuariaID    uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:ux_asignacion"`
	Contestada   bool       `gorm:"not null;default:false"`
	OperationID  *uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	EnvioID      *uuid.UUID `gorm:"type:uuid"`
	ContestadaEn *time.Time `gorm:"type:timestamptz"`
	CreadoEn     time.Time  `gorm:"not null;default:now();<-:create"`
}

func (AsignacionEncuesta) TableName() string { return "encuestas.asignaciones" }
