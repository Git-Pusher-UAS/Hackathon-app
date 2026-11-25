package gormrepo

import (
	"context"
	"crypto/sha256"
	"strconv"
	"time"

	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// auditChainLock serializa la construcción de la cadena de auditoría.
const auditChainLock int64 = 0x53414921 // "SAI!"

// ---------------- A36 · auditoria.eventos ----------------

type eventoAuditoriaRepo struct{ db *gorm.DB }

// Append asigna secuencia, hash_anterior y hash_actual bajo un candado de asesoría
// que serializa el encadenamiento, y luego inserta el evento.
//
// El formato del hash es una decisión de implementación (la interfaz no lo fija):
// SHA-256 sobre hash_anterior || secuencia || actor || acción || tipo_recurso ||
// recurso || resultado || ocurrido_en || ip || request_id.
func (r *eventoAuditoriaRepo) Append(ctx context.Context, e *models.EventoAuditoria) error {
	return translate(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", auditChainLock).Error; err != nil {
			return err
		}

		var last struct {
			Secuencia  int64  `gorm:"column:secuencia"`
			HashActual []byte `gorm:"column:hash_actual"`
		}
		res := tx.Raw(
			"SELECT secuencia, hash_actual FROM auditoria.eventos ORDER BY secuencia DESC LIMIT 1",
		).Scan(&last)
		if res.Error != nil {
			return res.Error
		}

		prev := make([]byte, sha256.Size)
		if res.RowsAffected > 0 {
			e.Secuencia = last.Secuencia + 1
			prev = last.HashActual
		} else {
			e.Secuencia = 1
		}

		e.HashAnterior = prev
		e.HashActual = auditHash(prev, e)
		return tx.Create(e).Error
	}))
}

func auditHash(prev []byte, e *models.EventoAuditoria) []byte {
	h := sha256.New()
	h.Write(prev)
	write := func(s string) { _, _ = h.Write([]byte(s)) }

	write(strconv.FormatInt(e.Secuencia, 10))
	write("|")
	if e.ActorID != nil {
		write(e.ActorID.String())
	}
	write("|")
	write(e.Accion)
	write("|")
	write(e.TipoRecurso)
	write("|")
	if e.RecursoID != nil {
		write(e.RecursoID.String())
	}
	write("|")
	write(string(e.Resultado))
	write("|")
	write(e.OcurridoEn.UTC().Format(time.RFC3339Nano))
	write("|")
	if e.IPOrigen != nil {
		write(*e.IPOrigen)
	}
	write("|")
	write(e.RequestID.String())

	return h.Sum(nil)
}

// Ultimo devuelve el último evento de la cadena con el candado tomado.
func (r *eventoAuditoriaRepo) Ultimo(ctx context.Context) (*models.EventoAuditoria, error) {
	var out models.EventoAuditoria
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", auditChainLock).Error; err != nil {
			return err
		}
		return tx.Order("secuencia DESC").First(&out).Error
	})
	if err != nil {
		return nil, translate(err)
	}
	return &out, nil
}

func (r *eventoAuditoriaRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.EventoAuditoria, error) {
	return first[models.EventoAuditoria](ctx, r.db, "id = ?", id)
}

func (r *eventoAuditoriaRepo) List(ctx context.Context, f repository.AuditoriaFilter, page repository.PageRequest) (repository.Page[models.EventoAuditoria], error) {
	q := r.db.WithContext(ctx).Model(&models.EventoAuditoria{})
	if f.ActorID != nil {
		q = q.Where("actor_id = ?", *f.ActorID)
	}
	if f.Accion != "" {
		q = q.Where("accion = ?", f.Accion)
	}
	if f.TipoRecurso != "" {
		q = q.Where("tipo_recurso = ?", f.TipoRecurso)
	}
	if f.RecursoID != nil {
		q = q.Where("recurso_id = ?", *f.RecursoID)
	}
	if f.Resultado != nil {
		q = q.Where("resultado = ?", *f.Resultado)
	}
	q = applyTimeRange(q, f.Rango, "ocurrido_en")

	limit := pageLimit(page)
	q = applyPage(q, page, "ocurrido_en", "id", true)

	var rows []models.EventoAuditoria
	if err := q.Find(&rows).Error; err != nil {
		return repository.Page[models.EventoAuditoria]{}, translate(err)
	}
	rows, next := cutPage(rows, limit,
		func(m models.EventoAuditoria) time.Time { return m.OcurridoEn },
		func(m models.EventoAuditoria) uuid.UUID { return m.ID },
	)
	return repository.Page[models.EventoAuditoria]{Items: rows, NextCursor: next}, nil
}

func (r *eventoAuditoriaRepo) Contar(ctx context.Context, actorID uuid.UUID, acciones []string, rango repository.TimeRange) (int, error) {
	q := r.db.WithContext(ctx).Model(&models.EventoAuditoria{}).Where("actor_id = ?", actorID)
	if len(acciones) > 0 {
		q = q.Where("accion IN ?", acciones)
	}
	q = applyTimeRange(q, rango, "ocurrido_en")

	var n int64
	if err := q.Count(&n).Error; err != nil {
		return 0, translate(err)
	}
	return int(n), nil
}

func (r *eventoAuditoriaRepo) RangoSecuencia(ctx context.Context, desdeExclusivo, hastaInclusivo int64) ([]models.EventoAuditoria, error) {
	return findAll[models.EventoAuditoria](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.Where("secuencia > ? AND secuencia <= ?", desdeExclusivo, hastaInclusivo).
			Order("secuencia ASC")
	})
}

// ---------------- A37 · auditoria.alertas ----------------

type alertaAuditoriaRepo struct{ db *gorm.DB }

func (r *alertaAuditoriaRepo) CreateIfAbsent(ctx context.Context, a *models.AlertaAuditoria) (bool, error) {
	res := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(a)
	if res.Error != nil {
		return false, translate(res.Error)
	}
	return res.RowsAffected > 0, nil
}

func (r *alertaAuditoriaRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.AlertaAuditoria, error) {
	return first[models.AlertaAuditoria](ctx, r.db, "id = ?", id)
}

func (r *alertaAuditoriaRepo) List(ctx context.Context, f repository.AlertaFilter, page repository.PageRequest) (repository.Page[models.AlertaAuditoria], error) {
	q := r.db.WithContext(ctx).Model(&models.AlertaAuditoria{})
	if f.ActorObservadoID != nil {
		q = q.Where("actor_observado_id = ?", *f.ActorObservadoID)
	}
	if f.Nivel != nil {
		q = q.Where("nivel = ?", *f.Nivel)
	}
	if f.Estado != nil {
		q = q.Where("estado = ?", *f.Estado)
	}
	q = applyTimeRange(q, f.Rango, "creado_en")

	limit := pageLimit(page)
	q = applyPage(q, page, "creado_en", "id", true)

	var rows []models.AlertaAuditoria
	if err := q.Find(&rows).Error; err != nil {
		return repository.Page[models.AlertaAuditoria]{}, translate(err)
	}
	rows, next := cutPage(rows, limit,
		func(m models.AlertaAuditoria) time.Time { return m.CreadoEn },
		func(m models.AlertaAuditoria) uuid.UUID { return m.ID },
	)
	return repository.Page[models.AlertaAuditoria]{Items: rows, NextCursor: next}, nil
}

func (r *alertaAuditoriaRepo) MarcarRevisada(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Model(&models.AlertaAuditoria{}).
		Where("id = ? AND estado = ?", id, models.AlertaAbierta).
		Update("estado", models.AlertaRevisada)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// ---------------- A38 · ayuda.eventos_locales ----------------

type eventoLocalAyudaRepo struct{ db *gorm.DB }

func (r *eventoLocalAyudaRepo) Create(ctx context.Context, e *models.EventoLocalAyuda) error {
	return translate(r.db.WithContext(ctx).Create(e).Error)
}

func (r *eventoLocalAyudaRepo) GetByOperacionID(ctx context.Context, operationID uuid.UUID) (*models.EventoLocalAyuda, error) {
	return first[models.EventoLocalAyuda](ctx, r.db, "operacion_id = ?", operationID)
}

func (r *eventoLocalAyudaRepo) ListByPersonal(ctx context.Context, personalID uuid.UUID, page repository.PageRequest) (repository.Page[models.EventoLocalAyuda], error) {
	q := r.db.WithContext(ctx).Model(&models.EventoLocalAyuda{}).
		Where("personal_id = ?", personalID)

	limit := pageLimit(page)
	q = applyPage(q, page, "creado_en", "id", true)

	var rows []models.EventoLocalAyuda
	if err := q.Find(&rows).Error; err != nil {
		return repository.Page[models.EventoLocalAyuda]{}, translate(err)
	}
	rows, next := cutPage(rows, limit,
		func(m models.EventoLocalAyuda) time.Time { return m.CreadoEn },
		func(m models.EventoLocalAyuda) uuid.UUID { return m.ID },
	)
	return repository.Page[models.EventoLocalAyuda]{Items: rows, NextCursor: next}, nil
}

// ---------------- analitica.v_resumen_mvp ----------------

type resumenRepo struct{ db *gorm.DB }

func (r *resumenRepo) Consultar(ctx context.Context, f repository.ResumenFiltro) ([]models.ResumenFila, error) {
	q := r.db.WithContext(ctx).Model(&models.ResumenFila{})
	if f.SectorID != nil {
		q = q.Where("sector_id = ?", *f.SectorID)
	}
	if f.TipoCasoID != nil {
		q = q.Where("tipo_caso_id = ?", *f.TipoCasoID)
	}
	q = applyTimeRange(q, f.Rango, "periodo")

	var rows []models.ResumenFila
	if err := q.Find(&rows).Error; err != nil {
		return nil, translate(err)
	}
	return rows, nil
}

// ---------------- analitica.v_mapa_mvp ----------------

type mapaRepo struct{ db *gorm.DB }

func (r *mapaRepo) Consultar(ctx context.Context, f repository.MapaFiltro) ([]models.MapaCelda, error) {
	q := r.db.WithContext(ctx).Model(&models.MapaCelda{})
	if f.TipoCasoID != nil {
		q = q.Where("tipo_caso_id = ?", *f.TipoCasoID)
	}
	if f.Periodo != nil {
		q = q.Where("periodo = ?", *f.Periodo)
	}

	var rows []models.MapaCelda
	if err := q.Find(&rows).Error; err != nil {
		return nil, translate(err)
	}
	return rows, nil
}
