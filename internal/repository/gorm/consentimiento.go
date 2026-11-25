package gormrepo

import (
	"context"
	"time"

	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---------------- A30 · consentimiento.decisiones ----------------

type decisionRepo struct{ db *gorm.DB }

func (r *decisionRepo) Create(ctx context.Context, d *models.Decision) error {
	return translate(r.db.WithContext(ctx).Create(d).Error)
}

func (r *decisionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Decision, error) {
	return first[models.Decision](ctx, r.db, "id = ?", id)
}

func (r *decisionRepo) GetByOperacionID(ctx context.Context, operationID uuid.UUID) (*models.Decision, error) {
	return first[models.Decision](ctx, r.db, "operacion_id = ?", operationID)
}

func (r *decisionRepo) SiguienteOrden(ctx context.Context, personaID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Raw(
		"SELECT COALESCE(MAX(orden_persona), 0) + 1 FROM consentimiento.decisiones WHERE persona_id = ?",
		personaID,
	).Scan(&n).Error
	return n, translate(err)
}

func (r *decisionRepo) ListByPersona(ctx context.Context, personaID uuid.UUID, page repository.PageRequest) (repository.Page[models.Decision], error) {
	q := r.db.WithContext(ctx).Model(&models.Decision{}).
		Where("persona_id = ?", personaID)

	limit := pageLimit(page)
	q = applyPage(q, page, "creado_en", "id", true)

	var rows []models.Decision
	if err := q.Find(&rows).Error; err != nil {
		return repository.Page[models.Decision]{}, translate(err)
	}
	rows, next := cutPage(rows, limit,
		func(m models.Decision) time.Time { return m.CreadoEn },
		func(m models.Decision) uuid.UUID { return m.ID },
	)
	return repository.Page[models.Decision]{Items: rows, NextCursor: next}, nil
}

// ---------------- A31 · consentimiento.decision_propositos ----------------

type decisionPropositoRepo struct{ db *gorm.DB }

func (r *decisionPropositoRepo) CreateMany(ctx context.Context, items []models.DecisionProposito) error {
	if len(items) == 0 {
		return nil
	}
	return translate(r.db.WithContext(ctx).Create(&items).Error)
}

func (r *decisionPropositoRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.DecisionProposito, error) {
	return first[models.DecisionProposito](ctx, r.db, "id = ?", id)
}

func (r *decisionPropositoRepo) ListByDecision(ctx context.Context, decisionID uuid.UUID) ([]models.DecisionProposito, error) {
	return findAll[models.DecisionProposito](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.Where("decision_id = ?", decisionID).Order("creado_en ASC")
	})
}

func (r *decisionPropositoRepo) ListByDecisiones(ctx context.Context, decisionIDs []uuid.UUID) ([]models.DecisionProposito, error) {
	if len(decisionIDs) == 0 {
		return []models.DecisionProposito{}, nil
	}
	return findAll[models.DecisionProposito](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.Where("decision_id IN ?", decisionIDs).Order("creado_en ASC")
	})
}

// ---------------- A32 · consentimiento.revocaciones ----------------

type revocacionRepo struct{ db *gorm.DB }

func (r *revocacionRepo) Create(ctx context.Context, rev *models.Revocacion) error {
	return translate(r.db.WithContext(ctx).Create(rev).Error)
}

func (r *revocacionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Revocacion, error) {
	return first[models.Revocacion](ctx, r.db, "id = ?", id)
}

func (r *revocacionRepo) ListByPersona(ctx context.Context, personaID uuid.UUID, page repository.PageRequest) (repository.Page[models.Revocacion], error) {
	q := r.db.WithContext(ctx).Model(&models.Revocacion{}).
		Where("persona_id = ?", personaID)

	limit := pageLimit(page)
	q = applyPage(q, page, "creado_en", "id", true)

	var rows []models.Revocacion
	if err := q.Find(&rows).Error; err != nil {
		return repository.Page[models.Revocacion]{}, translate(err)
	}
	rows, next := cutPage(rows, limit,
		func(m models.Revocacion) time.Time { return m.CreadoEn },
		func(m models.Revocacion) uuid.UUID { return m.ID },
	)
	return repository.Page[models.Revocacion]{Items: rows, NextCursor: next}, nil
}

// ---------------- A33 · consentimiento.revocacion_propositos ----------------

type revocacionPropositoRepo struct{ db *gorm.DB }

func (r *revocacionPropositoRepo) CreateMany(ctx context.Context, items []models.RevocacionProposito) error {
	if len(items) == 0 {
		return nil
	}
	return translate(r.db.WithContext(ctx).Create(&items).Error)
}

func (r *revocacionPropositoRepo) ListByRevocacion(ctx context.Context, revocacionID uuid.UUID) ([]models.RevocacionProposito, error) {
	return findAll[models.RevocacionProposito](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.Where("revocacion_id = ?", revocacionID).Order("creado_en ASC")
	})
}

func (r *revocacionPropositoRepo) ExistePorDecisionProposito(ctx context.Context, decisionPropositoID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.RevocacionProposito{}).
		Where("decision_proposito_id = ?", decisionPropositoID).
		Count(&n).Error
	if err != nil {
		return false, translate(err)
	}
	return n > 0, nil
}

// ---------------- consentimiento.v_estado_proposito ----------------

type estadoPropositoRepo struct{ db *gorm.DB }

func (r *estadoPropositoRepo) ListByPersona(ctx context.Context, personaID uuid.UUID) ([]models.EstadoProposito, error) {
	return findAll[models.EstadoProposito](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.Where("persona_id = ?", personaID).Order("decidido_en ASC")
	})
}

func (r *estadoPropositoRepo) Get(ctx context.Context, personaID, propositoID uuid.UUID) (*models.EstadoProposito, error) {
	return first[models.EstadoProposito](ctx, r.db, "persona_id = ? AND proposito_id = ?", personaID, propositoID)
}

func (r *estadoPropositoRepo) EstaVigente(ctx context.Context, personaID, propositoID uuid.UUID, at time.Time) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.EstadoProposito{}).
		Where("persona_id = ? AND proposito_id = ? AND vigente = ? AND decidido_en <= ?",
			personaID, propositoID, true, at).
		Count(&n).Error
	if err != nil {
		return false, translate(err)
	}
	return n > 0, nil
}
