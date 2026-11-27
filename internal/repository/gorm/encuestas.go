package gormrepo

import (
	"context"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type encuestaRepo struct{ db *gorm.DB }

func (r *encuestaRepo) ListConPreguntas(ctx context.Context) ([]models.Encuesta, error) {
	var out []models.Encuesta
	err := r.db.WithContext(ctx).Preload("Preguntas", func(q *gorm.DB) *gorm.DB { return q.Order("orden ASC") }).Order("creado_en DESC").Find(&out).Error
	return out, translate(err)
}

func (r *encuestaRepo) GetByIDWithPreguntas(ctx context.Context, id uuid.UUID) (*models.Encuesta, error) {
	var out models.Encuesta
	err := r.db.WithContext(ctx).Preload("Preguntas", func(q *gorm.DB) *gorm.DB { return q.Order("orden ASC") }).First(&out, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &out, nil
}

func (r *encuestaRepo) ListPendingByUsuaria(ctx context.Context, usuariaID uuid.UUID) ([]models.Encuesta, error) {
	var out []models.Encuesta
	err := r.db.WithContext(ctx).
		Joins("JOIN encuestas.asignaciones a ON a.encuesta_id = encuestas.encuestas.id").
		Where("a.usuaria_id = ? AND a.contestada = false AND encuestas.encuestas.activa = true", usuariaID).
		Preload("Preguntas", func(q *gorm.DB) *gorm.DB { return q.Order("orden ASC") }).
		Order("encuestas.encuestas.creado_en DESC").Find(&out).Error
	return out, translate(err)
}

func (r *encuestaRepo) GetAsignacion(ctx context.Context, encuestaID, usuariaID uuid.UUID) (*models.AsignacionEncuesta, error) {
	return first[models.AsignacionEncuesta](ctx, r.db, "encuesta_id = ? AND usuaria_id = ?", encuestaID, usuariaID)
}

func (r *encuestaRepo) GetAsignacionByOperationID(ctx context.Context, operationID uuid.UUID) (*models.AsignacionEncuesta, error) {
	return first[models.AsignacionEncuesta](ctx, r.db, "operation_id = ?", operationID)
}

func (r *encuestaRepo) SaveAsignacion(ctx context.Context, a *models.AsignacionEncuesta) error {
	return translate(r.db.WithContext(ctx).Save(a).Error)
}

func (r *encuestaRepo) CreateRespuestas(ctx context.Context, rows []models.RespuestaEncuesta) error {
	if len(rows) == 0 {
		return nil
	}
	return translate(r.db.WithContext(ctx).Create(&rows).Error)
}

func (r *encuestaRepo) Envios(ctx context.Context, encuestaID uuid.UUID) (int, error) {
	var n int
	err := r.db.WithContext(ctx).Raw(`SELECT count(DISTINCT envio_id) FROM encuestas.respuestas WHERE encuesta_id = ?`, encuestaID).Scan(&n).Error
	return n, translate(err)
}

func (r *encuestaRepo) Promedio(ctx context.Context, preguntaID uuid.UUID) (float64, int, error) {
	var row struct {
		Promedio float64
		Total    int
	}
	err := r.db.WithContext(ctx).Raw(`SELECT coalesce(avg(numero), 0)::float8 AS promedio, count(numero)::int AS total FROM encuestas.respuestas WHERE pregunta_id = ?`, preguntaID).Scan(&row).Error
	return row.Promedio, row.Total, translate(err)
}

func (r *encuestaRepo) ConteoOpciones(ctx context.Context, preguntaID uuid.UUID) (map[string]int, error) {
	var rows []struct {
		Texto string
		Total int
	}
	err := r.db.WithContext(ctx).Raw(`SELECT texto, count(*)::int AS total FROM encuestas.respuestas WHERE pregunta_id = ? AND texto IS NOT NULL GROUP BY texto`, preguntaID).Scan(&rows).Error
	out := make(map[string]int, len(rows))
	for _, x := range rows {
		out[x.Texto] = x.Total
	}
	return out, translate(err)
}
