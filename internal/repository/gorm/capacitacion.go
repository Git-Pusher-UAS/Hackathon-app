package gormrepo

import (
	"context"
	"time"

	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---------------- capacitacion.cursos ----------------

type cursoRepo struct{ db *gorm.DB }

func (r *cursoRepo) Create(ctx context.Context, c *models.Curso) error {
	return translate(r.db.WithContext(ctx).Create(c).Error)
}

func (r *cursoRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Curso, error) {
	return first[models.Curso](ctx, r.db, "id = ?", id)
}

func (r *cursoRepo) List(ctx context.Context, f repository.CursoFilter, page repository.PageRequest) (repository.Page[models.Curso], error) {
	q := r.db.WithContext(ctx).Model(&models.Curso{})
	if f.Tipo != nil {
		q = q.Where("tipo = ?", *f.Tipo)
	}
	if f.Desde != nil {
		q = q.Where("fecha >= ?", *f.Desde)
	}
	if f.Hasta != nil {
		q = q.Where("fecha <= ?", *f.Hasta)
	}
	limit := pageLimit(page)
	q = applyPage(q, page, "fecha", "id", false)
	var rows []models.Curso
	if err := q.Find(&rows).Error; err != nil {
		return repository.Page[models.Curso]{}, translate(err)
	}
	rows, next := cutPage(rows, limit,
		func(m models.Curso) time.Time { return m.Fecha },
		func(m models.Curso) uuid.UUID { return m.ID },
	)
	return repository.Page[models.Curso]{Items: rows, NextCursor: next}, nil
}

func (r *cursoRepo) Inscritas(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]int, error) {
	out := make(map[uuid.UUID]int, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		CursoID uuid.UUID
		Total   int
	}
	err := r.db.WithContext(ctx).Model(&models.Inscripcion{}).
		Select("curso_id, count(*) AS total").Where("curso_id IN ?", ids).Group("curso_id").Scan(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	for _, x := range rows {
		out[x.CursoID] = x.Total
	}
	return out, nil
}

// ---------------- capacitacion.inscripciones ----------------

type inscripcionRepo struct{ db *gorm.DB }

func (r *inscripcionRepo) ListByCurso(ctx context.Context, cursoID uuid.UUID) ([]models.Inscripcion, error) {
	return findAll[models.Inscripcion](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.Where("curso_id = ?", cursoID).Order("alias ASC, id ASC")
	})
}

func (r *inscripcionRepo) SetPresente(ctx context.Context, cursoID, id uuid.UUID, presente bool) error {
	res := r.db.WithContext(ctx).Model(&models.Inscripcion{}).
		Where("id = ? AND curso_id = ?", id, cursoID).Update("presente", presente)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// ---------------- capacitacion.visitas ----------------

type visitaRepo struct{ db *gorm.DB }

func (r *visitaRepo) Create(ctx context.Context, v *models.Visita) error {
	return translate(r.db.WithContext(ctx).Create(v).Error)
}

func (r *visitaRepo) List(ctx context.Context, estado *models.EstadoVisita, page repository.PageRequest) (repository.Page[models.Visita], error) {
	q := r.db.WithContext(ctx).Model(&models.Visita{})
	if estado != nil {
		q = q.Where("estado = ?", *estado)
	}
	limit := pageLimit(page)
	q = applyPage(q, page, "fecha", "id", false)
	var rows []models.Visita
	if err := q.Find(&rows).Error; err != nil {
		return repository.Page[models.Visita]{}, translate(err)
	}
	rows, next := cutPage(rows, limit,
		func(m models.Visita) time.Time { return m.Fecha },
		func(m models.Visita) uuid.UUID { return m.ID },
	)
	return repository.Page[models.Visita]{Items: rows, NextCursor: next}, nil
}
