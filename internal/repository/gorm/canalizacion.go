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

// ---------------- A34 · canalizacion.internas ----------------

type canalizacionRepo struct{ db *gorm.DB }

func (r *canalizacionRepo) Create(ctx context.Context, c *models.CanalizacionInterna) error {
	return translate(r.db.WithContext(ctx).Create(c).Error)
}

func (r *canalizacionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.CanalizacionInterna, error) {
	return first[models.CanalizacionInterna](ctx, r.db, "id = ?", id)
}

func (r *canalizacionRepo) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*models.CanalizacionInterna, error) {
	var out models.CanalizacionInterna
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&out).Error
	if err != nil {
		return nil, translate(err)
	}
	return &out, nil
}

func (r *canalizacionRepo) GetByOperacionID(ctx context.Context, operationID uuid.UUID) (*models.CanalizacionInterna, error) {
	return first[models.CanalizacionInterna](ctx, r.db, "operacion_id = ?", operationID)
}

func (r *canalizacionRepo) List(ctx context.Context, f repository.CanalizacionFilter, page repository.PageRequest) (repository.Page[models.CanalizacionInterna], error) {
	q := r.db.WithContext(ctx).Model(&models.CanalizacionInterna{})
	if len(f.DestinoIDs) > 0 {
		q = q.Where("destino_id IN ?", f.DestinoIDs)
	}
	if len(f.OrigenIDs) > 0 {
		q = q.Where("origen_id IN ?", f.OrigenIDs)
	}
	if f.EmitidaPor != nil {
		q = q.Where("emitida_por = ?", *f.EmitidaPor)
	}
	if f.ExpedienteID != nil {
		q = q.Where("expediente_id = ?", *f.ExpedienteID)
	}
	if len(f.Estados) > 0 {
		q = q.Where("estado IN ?", f.Estados)
	}

	limit := pageLimit(page)
	q = applyPage(q, page, "creado_en", "id", true)

	var rows []models.CanalizacionInterna
	if err := q.Find(&rows).Error; err != nil {
		return repository.Page[models.CanalizacionInterna]{}, translate(err)
	}
	rows, next := cutPage(rows, limit,
		func(m models.CanalizacionInterna) time.Time { return m.CreadoEn },
		func(m models.CanalizacionInterna) uuid.UUID { return m.ID },
	)
	return repository.Page[models.CanalizacionInterna]{Items: rows, NextCursor: next}, nil
}

func (r *canalizacionRepo) ExisteActiva(ctx context.Context, expedienteID, origenID, destinoID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.CanalizacionInterna{}).
		Where("expediente_id = ? AND origen_id = ? AND destino_id = ? AND estado IN ?",
			expedienteID, origenID, destinoID,
			[]models.EstadoCanalizacion{models.CanalizacionPendiente, models.CanalizacionAceptada}).
		Count(&n).Error
	if err != nil {
		return false, translate(err)
	}
	return n > 0, nil
}

func (r *canalizacionRepo) Update(ctx context.Context, c *models.CanalizacionInterna, expectedVersion int64) error {
	res := r.db.WithContext(ctx).Model(&models.CanalizacionInterna{}).
		Where("id = ? AND version = ?", c.ID, expectedVersion).
		Updates(map[string]any{
			"estado":                c.Estado,
			"asignacion_destino_id": c.AsignacionDestinoID,
			"version":               gorm.Expr("version + 1"),
			"actualizado_en":        time.Now(),
		})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrVersionMismatch
	}
	return nil
}

// ---------------- A35 · canalizacion.historial_internas ----------------

type historialCanalizacionRepo struct{ db *gorm.DB }

func (r *historialCanalizacionRepo) Append(ctx context.Context, h *models.HistorialCanalizacion) error {
	return translate(r.db.WithContext(ctx).Create(h).Error)
}

func (r *historialCanalizacionRepo) ListByCanalizacion(ctx context.Context, canalizacionID uuid.UUID) ([]models.HistorialCanalizacion, error) {
	return findAll[models.HistorialCanalizacion](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.Where("canalizacion_id = ?", canalizacionID).Order("version_resultante ASC")
	})
}
