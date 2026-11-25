package models

import (
	"time"

	"github.com/google/uuid"
)

// A30 · consentimiento.decisiones. No sobrescribible. UNIQUE(persona_id, orden_persona).
type Decision struct {
	ID               uuid.UUID               `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PersonaID        uuid.UUID               `gorm:"type:uuid;not null;uniqueIndex:ux_decision_orden"`
	AvisoID          uuid.UUID               `gorm:"type:uuid;not null"`
	ActorID          uuid.UUID               `gorm:"type:uuid;not null"`
	Decision         DecisionConsentimiento  `gorm:"type:varchar(15);not null"`
	Modalidad        ModalidadConsentimiento `gorm:"type:varchar(15);not null"`
	CapturadoEn      time.Time               `gorm:"not null"` // hora declarada
	EvidenciaCifrada []byte                  `gorm:"type:bytea"`
	OrdenPersona     int64                   `gorm:"not null;uniqueIndex:ux_decision_orden"` // asignado bajo bloqueo
	ClaveID          uuid.UUID               `gorm:"type:uuid;not null"`
	CreadoEn         time.Time               `gorm:"not null;default:now();<-:create"`
	OperacionID      *uuid.UUID              `gorm:"type:uuid;uniqueIndex"`
}

func (Decision) TableName() string { return "consentimiento.decisiones" }

// A31 · consentimiento.decision_propositos (UNIQUE decision_id, proposito_id; 1–10 por decisión).
type DecisionProposito struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DecisionID  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_decision_proposito"`
	PropositoID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_decision_proposito"`
	CreadoEn    time.Time `gorm:"not null;default:now();<-:create"`
}

func (DecisionProposito) TableName() string { return "consentimiento.decision_propositos" }

// A32 · consentimiento.revocaciones
type Revocacion struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PersonaID     uuid.UUID `gorm:"type:uuid;not null"`
	ActorID       uuid.UUID `gorm:"type:uuid;not null"`
	MotivoCifrado []byte    `gorm:"type:bytea"`
	OcurridoEn    time.Time `gorm:"not null"` // hora del servidor
	ClaveID       uuid.UUID `gorm:"type:uuid;not null"`
	CreadoEn      time.Time `gorm:"not null;default:now();<-:create"`
}

func (Revocacion) TableName() string { return "consentimiento.revocaciones" }

// A33 · consentimiento.revocacion_propositos (UNIQUE decision_proposito_id).
type RevocacionProposito struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RevocacionID        uuid.UUID `gorm:"type:uuid;not null"`
	DecisionPropositoID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	CreadoEn            time.Time `gorm:"not null;default:now();<-:create"`
}

func (RevocacionProposito) TableName() string { return "consentimiento.revocacion_propositos" }
