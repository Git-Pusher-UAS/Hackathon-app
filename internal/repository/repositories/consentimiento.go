package repository

import (
	"backend-hackathon/internal/models"
	"context"
	"time"

	"github.com/google/uuid"
)

// A30 · consentimiento.decisiones — inmutable
type DecisionRepository interface {
	Create(ctx context.Context, d *models.Decision) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Decision, error)
	GetByOperacionID(ctx context.Context, operacionID uuid.UUID) (*models.Decision, error)

	// SiguienteOrden devuelve max(orden_persona)+1. Debe llamarse con la fila de persona
	// bloqueada (PersonaRepository.GetByIDForUpdate).
	SiguienteOrden(ctx context.Context, personaID uuid.UUID) (int64, error)

	ListByPersona(ctx context.Context, personaID uuid.UUID, page PageRequest) (Page[models.Decision], error)
}

// A31 · consentimiento.decision_propositos
type DecisionPropositoRepository interface {
	// CreateMany inserta de 1 a 10 finalidades por decisión.
	CreateMany(ctx context.Context, items []models.DecisionProposito) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.DecisionProposito, error)
	ListByDecision(ctx context.Context, decisionID uuid.UUID) ([]models.DecisionProposito, error)
	ListByDecisiones(ctx context.Context, decisionIDs []uuid.UUID) ([]models.DecisionProposito, error)
}

// A32 · consentimiento.revocaciones — inmutable
type RevocacionRepository interface {
	Create(ctx context.Context, r *models.Revocacion) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Revocacion, error)
	ListByPersona(ctx context.Context, personaID uuid.UUID, page PageRequest) (Page[models.Revocacion], error)
}

// A33 · consentimiento.revocacion_propositos
type RevocacionPropositoRepository interface {
	// CreateMany devuelve ErrConflict si alguna finalidad de decisión ya fue revocada.
	CreateMany(ctx context.Context, items []models.RevocacionProposito) error
	ListByRevocacion(ctx context.Context, revocacionID uuid.UUID) ([]models.RevocacionProposito, error)

	// ExistePorDecisionProposito indica si esa finalidad aceptada ya fue revocada.
	ExistePorDecisionProposito(ctx context.Context, decisionPropositoID uuid.UUID) (bool, error)
}

// consentimiento.v_estado_proposito (vista de solo lectura)
type EstadoPropositoRepository interface {
	// ListByPersona devuelve el estado vigente de cada finalidad de la persona (API-029).
	ListByPersona(ctx context.Context, personaID uuid.UUID) ([]models.EstadoProposito, error)

	Get(ctx context.Context, personaID, propositoID uuid.UUID) (*models.EstadoProposito, error)

	// EstaVigente: última decisión aceptada y sin revocación. Un rechazo posterior desplaza
	// una aceptación previa; una revocación elimina la vigencia.
	EstaVigente(ctx context.Context, personaID, propositoID uuid.UUID, at time.Time) (bool, error)
}
