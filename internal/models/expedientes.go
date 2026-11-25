package models

import (
	"time"

	"github.com/google/uuid"
)

// A21 · expedientes.personas. Identidad civil, todo cifrado.
type Persona struct {
	ID                     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	NombreCifrado          []byte    `gorm:"type:bytea;not null"`
	CURPCifrada            []byte    `gorm:"column:curp_cifrada;type:bytea;not null"`
	FechaNacimientoCifrada []byte    `gorm:"type:bytea;not null"`
	TelefonoCifrado        []byte    `gorm:"type:bytea"`
	DomicilioCifrado       []byte    `gorm:"type:bytea"`
	SectorID               uuid.UUID `gorm:"type:uuid;not null"`
	RegistradaPor          uuid.UUID `gorm:"type:uuid;not null"`
	ClaveID                uuid.UUID `gorm:"type:uuid;not null"`
	Version                int64     `gorm:"not null;default:1"`
	ActualizadoEn          time.Time `gorm:"not null;default:now()"`
	CreadoEn               time.Time `gorm:"not null;default:now();<-:create"`
}

func (Persona) TableName() string { return "expedientes.personas" }

// A22 · expedientes.indices_curp (índice ciego).
// UNIQUE(version_hmac, valor_hmac) y UNIQUE(persona_id, version_hmac).
type IndiceCURP struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PersonaID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_indice_persona_version"`
	VersionHMAC int       `gorm:"column:version_hmac;not null;uniqueIndex:ux_indice_valor;uniqueIndex:ux_indice_persona_version"`
	ValorHMAC   []byte    `gorm:"column:valor_hmac;type:bytea;not null;uniqueIndex:ux_indice_valor"` // 32 bytes
	CreadoEn    time.Time `gorm:"not null;default:now();<-:create"`
}

func (IndiceCURP) TableName() string { return "expedientes.indices_curp" }

// A23 · expedientes.expedientes (1:1 con persona).
type Expediente struct {
	ID                  uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PersonaID           uuid.UUID        `gorm:"type:uuid;not null;uniqueIndex"`
	Alias               string           `gorm:"type:varchar(100);not null"`
	Estado              EstadoExpediente `gorm:"type:varchar(20);not null"`
	OrigenContacto      OrigenContacto   `gorm:"type:varchar(30);not null"`
	InstitucionOrigenID *uuid.UUID       `gorm:"type:uuid"`
	CreadoPor           uuid.UUID        `gorm:"type:uuid;not null"`
	CerradoEn           *time.Time       `gorm:"type:timestamptz"` // requerido si estado = cerrado
	Version             int64            `gorm:"not null;default:1"`
	ActualizadoEn       time.Time        `gorm:"not null;default:now()"`
	CreadoEn            time.Time        `gorm:"not null;default:now();<-:create"`
	OperacionID         uuid.UUID        `gorm:"type:uuid;not null;uniqueIndex"`
}

func (Expediente) TableName() string { return "expedientes.expedientes" }

// A24 · expedientes.episodios. UNIQUE(expediente_id, numero) y UNIQUE(id, expediente_id);
// índice único parcial por expediente WHERE cerrado_en IS NULL (un solo episodio activo).
type Episodio struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ExpedienteID uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:ux_episodio_numero"`
	Numero       int        `gorm:"not null;uniqueIndex:ux_episodio_numero"`
	TipoCasoID   uuid.UUID  `gorm:"type:uuid;not null"`
	AbiertoEn    time.Time  `gorm:"not null"`
	CerradoEn    *time.Time `gorm:"type:timestamptz"`
	CreadoEn     time.Time  `gorm:"not null;default:now();<-:create"`
}

func (Episodio) TableName() string { return "expedientes.episodios" }

// A25 · expedientes.historial_estados (inmutable). UNIQUE(expediente_id, version_resultante).
type HistorialEstado struct {
	ID                uuid.UUID         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ExpedienteID      uuid.UUID         `gorm:"type:uuid;not null;uniqueIndex:ux_historial_estado"`
	EstadoAnterior    *EstadoExpediente `gorm:"type:varchar(20)"` // nulo solo en el alta
	EstadoNuevo       EstadoExpediente  `gorm:"type:varchar(20);not null"`
	ActorID           uuid.UUID         `gorm:"type:uuid;not null"`
	MotivoCifrado     []byte            `gorm:"type:bytea"` // obligatorio al reabrir
	VersionResultante int64             `gorm:"not null;uniqueIndex:ux_historial_estado"`
	ClaveID           uuid.UUID         `gorm:"type:uuid;not null"`
	CreadoEn          time.Time         `gorm:"not null;default:now();<-:create"`
}

func (HistorialEstado) TableName() string { return "expedientes.historial_estados" }

// A26 · expedientes.asignaciones. Único parcial (expediente, personal, depto) WHERE fin IS NULL.
type Asignacion struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ExpedienteID   uuid.UUID  `gorm:"type:uuid;not null"`
	PersonalID     uuid.UUID  `gorm:"type:uuid;not null"`
	DepartamentoID uuid.UUID  `gorm:"type:uuid;not null"`
	AsignadaPor    uuid.UUID  `gorm:"type:uuid;not null"`
	Inicio         time.Time  `gorm:"not null"`
	Fin            *time.Time `gorm:"type:timestamptz"` // CHECK fin >= inicio
	MotivoCifrado  []byte     `gorm:"type:bytea;not null"`
	ClaveID        uuid.UUID  `gorm:"type:uuid;not null"`
	CreadoEn       time.Time  `gorm:"not null;default:now();<-:create"`
}

func (Asignacion) TableName() string { return "expedientes.asignaciones" }

// A27 · expedientes.notas. Solo INSERT: las correcciones se hacen por adición.
// FKs compuestas: (episodio_id, expediente_id) y
// (corrige_nota_id, expediente_id, departamento_id).
type Nota struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ExpedienteID   uuid.UUID  `gorm:"type:uuid;not null;<-:create"`
	EpisodioID     uuid.UUID  `gorm:"type:uuid;not null;<-:create"`
	DepartamentoID uuid.UUID  `gorm:"type:uuid;not null;<-:create"`
	AutorID        uuid.UUID  `gorm:"type:uuid;not null;<-:create"`
	TextoCifrado   []byte     `gorm:"type:bytea;not null;<-:create"`
	OcurridoEn     time.Time  `gorm:"not null;<-:create"` // hora declarada por el dispositivo
	CorrigeNotaID  *uuid.UUID `gorm:"type:uuid;<-:create"`
	OperacionID    uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex;<-:create"`
	ClaveID        uuid.UUID  `gorm:"type:uuid;not null;<-:create"`
	CreadoEn       time.Time  `gorm:"not null;default:now();<-:create"`
}

func (Nota) TableName() string { return "expedientes.notas" }

// A28 · expedientes.tokens_vinculacion
type TokenVinculacion struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ExpedienteID  uuid.UUID  `gorm:"type:uuid;not null"`
	SolicitanteID uuid.UUID  `gorm:"type:uuid;not null"`
	TokenHash     []byte     `gorm:"type:bytea;not null;uniqueIndex"`
	VenceEn       time.Time  `gorm:"not null"`
	ConsumidoEn   *time.Time `gorm:"type:timestamptz"`
	CreadoEn      time.Time  `gorm:"not null;default:now();<-:create"`
}

func (TokenVinculacion) TableName() string { return "expedientes.tokens_vinculacion" }

// A29 · expedientes.solicitudes_vinculacion
type SolicitudVinculacion struct {
	ID            uuid.UUID                  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TokenID       uuid.UUID                  `gorm:"type:uuid;not null;uniqueIndex"`
	SolicitanteID uuid.UUID                  `gorm:"type:uuid;not null"`
	ExpedienteID  uuid.UUID                  `gorm:"type:uuid;not null"`
	MotivoCifrado []byte                     `gorm:"type:bytea;not null"`
	Estado        EstadoSolicitudVinculacion `gorm:"type:varchar(15);not null"`
	ResueltaPor   *uuid.UUID                 `gorm:"type:uuid"`
	ResueltaEn    *time.Time                 `gorm:"type:timestamptz"` // par inseparable con resuelta_por
	AsignacionID  *uuid.UUID                 `gorm:"type:uuid"`        // requerida al aprobar
	ClaveID       uuid.UUID                  `gorm:"type:uuid;not null"`
	Version       int64                      `gorm:"not null;default:1"`
	ActualizadoEn time.Time                  `gorm:"not null;default:now()"`
	CreadoEn      time.Time                  `gorm:"not null;default:now();<-:create"`
}

func (SolicitudVinculacion) TableName() string { return "expedientes.solicitudes_vinculacion" }
