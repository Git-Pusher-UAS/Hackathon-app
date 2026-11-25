package gormrepo

import (
	"context"
	"time"

	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---------------- A19 · seguridad.claves_registro ----------------

type claveRegistroRepo struct{ db *gorm.DB }

func (r *claveRegistroRepo) Create(ctx context.Context, c *models.ClaveRegistro) error {
	return translate(r.db.WithContext(ctx).Create(c).Error)
}

func (r *claveRegistroRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.ClaveRegistro, error) {
	return first[models.ClaveRegistro](ctx, r.db, "id = ?", id)
}

func (r *claveRegistroRepo) GetByRegistro(ctx context.Context, modulo, tipoRegistro string, registroID uuid.UUID) (*models.ClaveRegistro, error) {
	return first[models.ClaveRegistro](ctx, r.db,
		"modulo = ? AND tipo_registro = ? AND registro_id = ?",
		modulo, tipoRegistro, registroID,
	)
}

func (r *claveRegistroRepo) ListParaRotar(ctx context.Context, versionKEKObjetivo int, limit int) ([]models.ClaveRegistro, error) {
	if limit <= 0 {
		limit = 100
	}
	return findAll[models.ClaveRegistro](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.Where("version_kek < ?", versionKEKObjetivo).
			Order("creado_en ASC").
			Limit(limit)
	})
}

func (r *claveRegistroRepo) ActualizarEnvoltura(ctx context.Context, id uuid.UUID, dekEnvuelta, nonce []byte, versionKEK int, rotadaEn time.Time) error {
	res := r.db.WithContext(ctx).Model(&models.ClaveRegistro{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"dek_envuelta":    dekEnvuelta,
			"nonce_envoltura": nonce,
			"version_kek":     versionKEK,
			"rotada_en":       rotadaEn,
		})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// ---------------- A20 · seguridad.operaciones ----------------

type operacionRepo struct{ db *gorm.DB }

func (r *operacionRepo) Create(ctx context.Context, op *models.Operacion) error {
	return translate(r.db.WithContext(ctx).Create(op).Error)
}

func (r *operacionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Operacion, error) {
	return first[models.Operacion](ctx, r.db, "id = ?", id)
}

func (r *operacionRepo) Find(ctx context.Context, actorID uuid.UUID, tipoOperacion string, operationID uuid.UUID) (*models.Operacion, error) {
	return first[models.Operacion](ctx, r.db,
		"actor_id = ? AND tipo_operacion = ? AND operation_id = ?",
		actorID, tipoOperacion, operationID,
	)
}

func (r *operacionRepo) FindForUpdate(ctx context.Context, actorID uuid.UUID, tipoOperacion string, operationID uuid.UUID) (*models.Operacion, error) {
	var out models.Operacion
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("actor_id = ? AND tipo_operacion = ? AND operation_id = ?", actorID, tipoOperacion, operationID).
		First(&out).Error
	if err != nil {
		return nil, translate(err)
	}
	return &out, nil
}

func (r *operacionRepo) PurgarResultados(ctx context.Context, before time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Model(&models.Operacion{}).
		Where("resultado_vence_en <= ? AND resultado_cifrado IS NOT NULL", before).
		Update("resultado_cifrado", nil)
	return res.RowsAffected, translate(res.Error)
}
