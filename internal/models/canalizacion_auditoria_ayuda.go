package models

import (
	"time"

	"github.com/google/uuid"
)

// A34 · canalizacion.internas.
type CanalizacionInterna struct {
	ID                  uuid.UUID          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ExpedienteID        uuid.UUID          `gorm:"type:uuid;not null"`
	OrigenID            uuid.UUID          `gorm:"type:uuid;not null"`
	Tipo                TipoCanalizacion   `gorm:"type:varchar(10);not null;default:interna"`
	DestinoID           *uuid.UUID         `gorm:"type:uuid"`
	InstitucionID       *uuid.UUID         `gorm:"type:uuid"`
	EmitidaPor          uuid.UUID          `gorm:"type:uuid;not null"`
	ResumenCifrado      []byte             `gorm:"type:bytea;not null"`
	Estado              EstadoCanalizacion `gorm:"type:varchar(20);not null"`
	AsignacionDestinoID *uuid.UUID         `gorm:"type:uuid"`
	OperacionID         uuid.UUID          `gorm:"type:uuid;not null;uniqueIndex"`
	ClaveID             uuid.UUID          `gorm:"type:uuid;not null"`
	Version             int64              `gorm:"not null;default:1"`
	ActualizadoEn       time.Time          `gorm:"not null;default:now()"`
	CreadoEn            time.Time          `gorm:"not null;default:now();<-:create"`
}

func (CanalizacionInterna) TableName() string { return "canalizacion.internas" }

// A35 · canalizacion.historial_internas. UNIQUE(canalizacion_id, version_resultante).
type HistorialCanalizacion struct {
	ID                uuid.UUID           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CanalizacionID    uuid.UUID           `gorm:"type:uuid;not null;uniqueIndex:ux_historial_canalizacion"`
	ActorID           uuid.UUID           `gorm:"type:uuid;not null"`
	EstadoAnterior    *EstadoCanalizacion `gorm:"type:varchar(20)"` // nulo solo en la creación
	EstadoNuevo       EstadoCanalizacion  `gorm:"type:varchar(20);not null"`
	MotivoCifrado     []byte              `gorm:"type:bytea"` // obligatorio al rechazar/cancelar
	VersionResultante int64               `gorm:"not null;uniqueIndex:ux_historial_canalizacion"`
	ClaveID           uuid.UUID           `gorm:"type:uuid;not null"`
	CreadoEn          time.Time           `gorm:"not null;default:now();<-:create"`
}

func (HistorialCanalizacion) TableName() string { return "canalizacion.historial_internas" }

// A36 · auditoria.eventos. Append-only encadenado por hash.
// recurso_id NO tiene FK a propósito (puede auditarse un UUID inexistente).
type EventoAuditoria struct {
	ID           uuid.UUID          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Secuencia    int64              `gorm:"not null;uniqueIndex"`
	ActorID      *uuid.UUID         `gorm:"type:uuid"` // nulo solo en acción anónima
	Accion       string             `gorm:"type:varchar(100);not null"`
	TipoRecurso  string             `gorm:"type:varchar(80);not null"`
	RecursoID    *uuid.UUID         `gorm:"type:uuid"`
	Resultado    ResultadoAuditoria `gorm:"type:varchar(20);not null"`
	OcurridoEn   time.Time          `gorm:"not null"`
	IPOrigen     *string            `gorm:"type:inet"`
	RequestID    uuid.UUID          `gorm:"type:uuid;not null"`
	HashAnterior []byte             `gorm:"type:bytea;not null"` // 32 bytes
	HashActual   []byte             `gorm:"type:bytea;not null;uniqueIndex"`
	CreadoEn     time.Time          `gorm:"not null;default:now();<-:create"`
}

func (EventoAuditoria) TableName() string { return "auditoria.eventos" }

// A37 · auditoria.alertas. UNIQUE(actor_observado_id, tipo, ventana_inicio, ventana_fin).
type AlertaAuditoria struct {
	ID                 uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ActorObservadoID   uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:ux_alerta"`
	Tipo               string       `gorm:"type:varchar(40);not null;uniqueIndex:ux_alerta"`
	VentanaInicio      time.Time    `gorm:"not null;uniqueIndex:ux_alerta"`
	VentanaFin         time.Time    `gorm:"not null;uniqueIndex:ux_alerta"`
	Conteo             int          `gorm:"not null"`
	Umbral             int          `gorm:"not null"`
	Nivel              NivelAlerta  `gorm:"type:varchar(15);not null"`
	Estado             EstadoAlerta `gorm:"type:varchar(15);not null"`
	EventoDisparadorID *uuid.UUID   `gorm:"type:uuid"`
	CreadoEn           time.Time    `gorm:"not null;default:now();<-:create"`
}

func (AlertaAuditoria) TableName() string { return "auditoria.alertas" }

// A38 · ayuda.eventos_locales
type EventoLocalAyuda struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PersonalID  uuid.UUID  `gorm:"type:uuid;not null"`
	ContactoID  *uuid.UUID `gorm:"type:uuid"`                 // institución del directorio
	Accion      string     `gorm:"type:varchar(30);not null"` // dialer_opened
	OcurridoEn  time.Time  `gorm:"not null"`                  // fecha declarada (offline)
	OperacionID uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	CreadoEn    time.Time  `gorm:"not null;default:now();<-:create"`
}

func (EventoLocalAyuda) TableName() string { return "ayuda.eventos_locales" }
