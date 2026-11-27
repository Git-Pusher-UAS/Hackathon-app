package gormrepo

import (
	"context"
	"fmt"
	"strings"

	repository "backend-hackathon/internal/repository/repositories"

	"gorm.io/gorm"
)

type indicadoresRepo struct{ db *gorm.DB }

func indicadorWhere(alias string, f repository.IndicadoresFiltro) (string, []any) {
	clauses := []string{fmt.Sprintf("%s.creado_en >= ? AND %s.creado_en < ?", alias, alias)}
	args := []any{f.Rango.Desde, f.Rango.Hasta}
	if f.TipoCasoID != nil {
		clauses = append(clauses, fmt.Sprintf("%s.tipo_caso_id = ?", alias))
		args = append(args, *f.TipoCasoID)
	}
	if f.Estado != nil {
		clauses = append(clauses, fmt.Sprintf("%s.estado = ?", alias))
		args = append(args, *f.Estado)
	}
	return strings.Join(clauses, " AND "), args
}

func (r *indicadoresRepo) Expedientes(ctx context.Context, f repository.IndicadoresFiltro) (int, error) {
	where, args := indicadorWhere("e", f)
	var n int
	err := r.db.WithContext(ctx).Raw("SELECT count(*) FROM expedientes.expedientes e WHERE "+where, args...).Scan(&n).Error
	return n, translate(err)
}

func (r *indicadoresRepo) Canalizaciones(ctx context.Context, f repository.IndicadoresFiltro) (int, error) {
	where, args := indicadorWhere("e", f)
	q := `SELECT count(*) FROM canalizacion.internas c JOIN expedientes.expedientes e ON e.id = c.expediente_id WHERE ` + where
	var n int
	err := r.db.WithContext(ctx).Raw(q, args...).Scan(&n).Error
	return n, translate(err)
}

func (r *indicadoresRepo) PorMes(ctx context.Context, f repository.IndicadoresFiltro) ([]repository.Conteo, error) {
	where, args := indicadorWhere("e", f)
	q := `SELECT to_char(date_trunc('month', e.creado_en), 'YYYY-MM') AS etiqueta, count(*)::int AS valor
FROM expedientes.expedientes e WHERE ` + where + ` GROUP BY 1 ORDER BY 1`
	var out []repository.Conteo
	err := r.db.WithContext(ctx).Raw(q, args...).Scan(&out).Error
	return out, translate(err)
}

func (r *indicadoresRepo) PorOrigen(ctx context.Context, f repository.IndicadoresFiltro) ([]repository.Conteo, error) {
	where, args := indicadorWhere("e", f)
	q := `SELECT e.origen_contacto AS etiqueta, count(*)::int AS valor
FROM expedientes.expedientes e WHERE ` + where + ` GROUP BY 1 ORDER BY 2 DESC, 1`
	var out []repository.Conteo
	err := r.db.WithContext(ctx).Raw(q, args...).Scan(&out).Error
	return out, translate(err)
}

func (r *indicadoresRepo) PorArea(ctx context.Context, f repository.IndicadoresFiltro) ([]repository.Conteo, error) {
	where, args := indicadorWhere("e", f)
	q := `SELECT d.nombre AS etiqueta, count(DISTINCT e.id)::int AS valor
FROM expedientes.expedientes e
JOIN expedientes.asignaciones a ON a.expediente_id = e.id
JOIN catalogos.departamentos d ON d.id = a.departamento_id
WHERE ` + where + ` GROUP BY 1 ORDER BY 2 DESC, 1`
	var out []repository.Conteo
	err := r.db.WithContext(ctx).Raw(q, args...).Scan(&out).Error
	return out, translate(err)
}

func (r *indicadoresRepo) PorSector(ctx context.Context, f repository.IndicadoresFiltro) ([]repository.Conteo, error) {
	where, args := indicadorWhere("e", f)
	q := `SELECT s.id::text AS etiqueta, count(e.id)::int AS valor
FROM catalogos.sectores s
LEFT JOIN expedientes.personas p ON p.sector_id = s.id
LEFT JOIN expedientes.expedientes e ON e.persona_id = p.id AND ` + where + `
WHERE s.activo
GROUP BY s.id ORDER BY s.id::text`
	var out []repository.Conteo
	err := r.db.WithContext(ctx).Raw(q, args...).Scan(&out).Error
	return out, translate(err)
}
