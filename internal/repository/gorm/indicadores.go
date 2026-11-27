package gormrepo

import (
	"context"

	repository "backend-hackathon/internal/repository/repositories"

	"gorm.io/gorm"
)

// ---------------- agregados del tablero (§13.2) ----------------

type indicadoresRepo struct{ db *gorm.DB }

func (r *indicadoresRepo) count(ctx context.Context, sql string, rg repository.TimeRange) (int, error) {
	var n int
	err := r.db.WithContext(ctx).Raw(sql, rg.Desde, rg.Hasta).Scan(&n).Error
	return n, translate(err)
}

func (r *indicadoresRepo) conteos(ctx context.Context, sql string, rg repository.TimeRange) ([]repository.Conteo, error) {
	var out []repository.Conteo
	err := r.db.WithContext(ctx).Raw(sql, rg.Desde, rg.Hasta).Scan(&out).Error
	return out, translate(err)
}

func (r *indicadoresRepo) Expedientes(ctx context.Context, rg repository.TimeRange) (int, error) {
	return r.count(ctx, `SELECT count(*) FROM expedientes.expedientes WHERE creado_en >= ? AND creado_en < ?`, rg)
}

func (r *indicadoresRepo) Canalizaciones(ctx context.Context, rg repository.TimeRange) (int, error) {
	return r.count(ctx, `SELECT count(*) FROM canalizacion.internas WHERE creado_en >= ? AND creado_en < ?`, rg)
}

func (r *indicadoresRepo) PorMes(ctx context.Context, rg repository.TimeRange) ([]repository.Conteo, error) {
	return r.conteos(ctx, `
		SELECT to_char(date_trunc('month', creado_en), 'YYYY-MM') AS etiqueta, count(*)::int AS valor
		FROM expedientes.expedientes WHERE creado_en >= ? AND creado_en < ?
		GROUP BY 1 ORDER BY 1`, rg)
}

func (r *indicadoresRepo) PorOrigen(ctx context.Context, rg repository.TimeRange) ([]repository.Conteo, error) {
	return r.conteos(ctx, `
		SELECT origen_contacto AS etiqueta, count(*)::int AS valor
		FROM expedientes.expedientes WHERE creado_en >= ? AND creado_en < ?
		GROUP BY 1 ORDER BY 2 DESC, 1`, rg)
}

func (r *indicadoresRepo) PorArea(ctx context.Context, rg repository.TimeRange) ([]repository.Conteo, error) {
	return r.conteos(ctx, `
		SELECT d.codigo AS etiqueta, count(DISTINCT e.id)::int AS valor
		FROM expedientes.expedientes e
		JOIN expedientes.asignaciones a ON a.expediente_id = e.id
		JOIN catalogos.departamentos d ON d.id = a.departamento_id
		WHERE e.creado_en >= ? AND e.creado_en < ?
		GROUP BY 1 ORDER BY 2 DESC, 1`, rg)
}

func (r *indicadoresRepo) PorSector(ctx context.Context, rg repository.TimeRange) ([]repository.Conteo, error) {
	return r.conteos(ctx, `
		SELECT s.ageb AS etiqueta, count(e.id)::int AS valor
		FROM catalogos.sectores s
		LEFT JOIN expedientes.personas p ON p.sector_id = s.id
		LEFT JOIN expedientes.expedientes e ON e.persona_id = p.id AND e.creado_en >= ? AND e.creado_en < ?
		WHERE s.ageb IS NOT NULL AND s.activo
		GROUP BY s.ageb ORDER BY s.ageb`, rg)
}
