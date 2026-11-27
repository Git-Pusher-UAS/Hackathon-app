package gormrepo

import (
	"context"
	"strings"
	"time"

	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type vacanteRepo struct{ db *gorm.DB }

func (r *vacanteRepo) Create(ctx context.Context, v *models.Vacante) error {
	return translate(r.db.WithContext(ctx).Create(v).Error)
}
func (r *vacanteRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Vacante, error) {
	return first[models.Vacante](ctx, r.db, "id = ?", id)
}
func (r *vacanteRepo) List(ctx context.Context, f repository.VacanteFilter, page repository.PageRequest) (repository.Page[models.Vacante], error) {
	q := r.db.WithContext(ctx).Model(&models.Vacante{})
	if f.Sector != "" {
		q = q.Where("lower(sector) = lower(?)", strings.TrimSpace(f.Sector))
	}
	if f.Horario != "" {
		q = q.Where("lower(horario) = lower(?)", strings.TrimSpace(f.Horario))
	}
	if f.TipoEmpleo != "" {
		q = q.Where("lower(tipo_empleo) = lower(?)", strings.TrimSpace(f.TipoEmpleo))
	}
	if f.Ubicacion != "" {
		q = q.Where("lower(ubicacion) LIKE lower(?)", "%"+strings.TrimSpace(f.Ubicacion)+"%")
	}
	limit := pageLimit(page)
	q = applyPage(q, page, "publicada_en", "id", true)
	var rows []models.Vacante
	if err := q.Find(&rows).Error; err != nil {
		return repository.Page[models.Vacante]{}, translate(err)
	}
	rows, next := cutPage(rows, limit, func(m models.Vacante) time.Time { return m.PublicadaEn }, func(m models.Vacante) uuid.UUID { return m.ID })
	return repository.Page[models.Vacante]{Items: rows, NextCursor: next}, nil
}
func (r *vacanteRepo) CountPostulaciones(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]int, error) {
	out := make(map[uuid.UUID]int, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		VacanteID uuid.UUID
		Total     int
	}
	err := r.db.WithContext(ctx).Model(&models.Postulacion{}).Select("vacante_id, count(*) as total").Where("vacante_id IN ?", ids).Group("vacante_id").Scan(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	for _, x := range rows {
		out[x.VacanteID] = x.Total
	}
	return out, nil
}

type postulacionRepo struct{ db *gorm.DB }

func (r *postulacionRepo) Create(ctx context.Context, p *models.Postulacion) error {
	return translate(r.db.WithContext(ctx).Create(p).Error)
}
func (r *postulacionRepo) Exists(ctx context.Context, vacanteID, usuariaID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.Postulacion{}).Where("vacante_id = ? AND usuaria_id = ?", vacanteID, usuariaID).Count(&n).Error
	return n > 0, translate(err)
}
func (r *postulacionRepo) ExistsMany(ctx context.Context, vacanteIDs []uuid.UUID, usuariaID uuid.UUID) (map[uuid.UUID]bool, error) {
	out := map[uuid.UUID]bool{}
	if len(vacanteIDs) == 0 {
		return out, nil
	}
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).Model(&models.Postulacion{}).Where("vacante_id IN ? AND usuaria_id = ?", vacanteIDs, usuariaID).Pluck("vacante_id", &ids).Error
	if err != nil {
		return nil, translate(err)
	}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}
