package gormrepo

import (
	"context"
	"errors"
	"time"

	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---------------- A21 · expedientes.personas ----------------

type personaRepo struct{ db *gorm.DB }

func (r *personaRepo) Create(ctx context.Context, p *models.Persona) error {
	return translate(r.db.WithContext(ctx).Create(p).Error)
}

func (r *personaRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Persona, error) {
	return first[models.Persona](ctx, r.db, "id = ?", id)
}

func (r *personaRepo) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*models.Persona, error) {
	var out models.Persona
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&out).Error
	if err != nil {
		return nil, translate(err)
	}
	return &out, nil
}

func (r *personaRepo) Update(ctx context.Context, p *models.Persona, expectedVersion int64) error {
	res := r.db.WithContext(ctx).Model(&models.Persona{}).
		Where("id = ? AND version = ?", p.ID, expectedVersion).
		Updates(map[string]any{
			"nombre_cifrado":           p.NombreCifrado,
			"curp_cifrada":             p.CURPCifrada,
			"fecha_nacimiento_cifrada": p.FechaNacimientoCifrada,
			"telefono_cifrado":         p.TelefonoCifrado,
			"domicilio_cifrado":        p.DomicilioCifrado,
			"sector_id":                p.SectorID,
			"clave_id":                 p.ClaveID,
			"version":                  gorm.Expr("version + 1"),
			"actualizado_en":           time.Now(),
		})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrVersionMismatch
	}
	return nil
}

// ---------------- A22 · expedientes.indices_curp ----------------

type indiceCURPRepo struct{ db *gorm.DB }

func (r *indiceCURPRepo) Create(ctx context.Context, idx *models.IndiceCURP) error {
	return translate(r.db.WithContext(ctx).Create(idx).Error)
}

func (r *indiceCURPRepo) BuscarPersona(ctx context.Context, candidatos []repository.IndiceCURPClave) (uuid.UUID, error) {
	for _, c := range candidatos {
		var idx models.IndiceCURP
		err := r.db.WithContext(ctx).
			Where("version_hmac = ? AND valor_hmac = ?", c.VersionHMAC, c.ValorHMAC).
			First(&idx).Error
		if err == nil {
			return idx.PersonaID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, translate(err)
		}
	}
	return uuid.Nil, repository.ErrNotFound
}

func (r *indiceCURPRepo) ListByPersona(ctx context.Context, personaID uuid.UUID) ([]models.IndiceCURP, error) {
	return findAll[models.IndiceCURP](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.Where("persona_id = ?", personaID).Order("version_hmac ASC")
	})
}

func (r *indiceCURPRepo) ListPersonasSinVersion(ctx context.Context, versionHMAC int, limit int) ([]uuid.UUID, error) {
	if limit <= 0 {
		limit = 500
	}
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).Raw(`
		SELECT p.id
		FROM expedientes.personas p
		LEFT JOIN expedientes.indices_curp i
		       ON i.persona_id = p.id AND i.version_hmac = ?
		WHERE i.id IS NULL
		ORDER BY p.creado_en ASC
		LIMIT ?`, versionHMAC, limit).Scan(&ids).Error
	return ids, translate(err)
}

func (r *indiceCURPRepo) CountPersonasSinVersion(ctx context.Context, versionHMAC int) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT count(*)
		FROM expedientes.personas p
		LEFT JOIN expedientes.indices_curp i
		       ON i.persona_id = p.id AND i.version_hmac = ?
		WHERE i.id IS NULL`, versionHMAC).Scan(&n).Error
	return n, translate(err)
}

func (r *indiceCURPRepo) DeleteVersion(ctx context.Context, versionHMAC int) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("version_hmac = ?", versionHMAC).
		Delete(&models.IndiceCURP{})
	return res.RowsAffected, translate(res.Error)
}

// ---------------- A23 · expedientes.expedientes ----------------

type expedienteRepo struct{ db *gorm.DB }

func (r *expedienteRepo) Create(ctx context.Context, e *models.Expediente) error {
	return translate(r.db.WithContext(ctx).Create(e).Error)
}

func (r *expedienteRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Expediente, error) {
	return first[models.Expediente](ctx, r.db, "id = ?", id)
}

func (r *expedienteRepo) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*models.Expediente, error) {
	var out models.Expediente
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&out).Error
	if err != nil {
		return nil, translate(err)
	}
	return &out, nil
}

func (r *expedienteRepo) GetByPersonaID(ctx context.Context, personaID uuid.UUID) (*models.Expediente, error) {
	return first[models.Expediente](ctx, r.db, "persona_id = ?", personaID)
}

func (r *expedienteRepo) GetByOperacionID(ctx context.Context, operationID uuid.UUID) (*models.Expediente, error) {
	return first[models.Expediente](ctx, r.db, "operacion_id = ?", operationID)
}

func (r *expedienteRepo) List(ctx context.Context, f repository.ExpedienteFilter, page repository.PageRequest) (repository.Page[models.Expediente], error) {
	q := r.db.WithContext(ctx).Model(&models.Expediente{})

	if f.AsignadoA != nil {
		q = q.Where(`EXISTS (
			SELECT 1 FROM expedientes.asignaciones a
			WHERE a.expediente_id = expedientes.expedientes.id
			  AND a.fin IS NULL AND a.personal_id = ?)`, *f.AsignadoA)
	}
	if f.DepartamentoID != nil {
		q = q.Where(`EXISTS (
			SELECT 1 FROM expedientes.asignaciones a
			WHERE a.expediente_id = expedientes.expedientes.id
			  AND a.fin IS NULL AND a.departamento_id = ?)`, *f.DepartamentoID)
	}
	if f.TipoCasoID != nil {
		q = q.Where(`EXISTS (
			SELECT 1 FROM expedientes.episodios ep
			WHERE ep.expediente_id = expedientes.expedientes.id
			  AND ep.cerrado_en IS NULL AND ep.tipo_caso_id = ?)`, *f.TipoCasoID)
	}
	if f.Estado != nil {
		q = q.Where("estado = ?", *f.Estado)
	}
	q = applyTimeRange(q, f.Actualizado, "actualizado_en")
	if f.Alias != "" {
		q = q.Where("alias ILIKE ?", "%"+f.Alias+"%")
	}

	limit := pageLimit(page)
	q = applyPage(q, page, "creado_en", "id", false)

	var rows []models.Expediente
	if err := q.Find(&rows).Error; err != nil {
		return repository.Page[models.Expediente]{}, translate(err)
	}
	rows, next := cutPage(rows, limit,
		func(m models.Expediente) time.Time { return m.CreadoEn },
		func(m models.Expediente) uuid.UUID { return m.ID },
	)
	return repository.Page[models.Expediente]{Items: rows, NextCursor: next}, nil
}

func (r *expedienteRepo) Update(ctx context.Context, e *models.Expediente, expectedVersion int64) error {
	res := r.db.WithContext(ctx).Model(&models.Expediente{}).
		Where("id = ? AND version = ?", e.ID, expectedVersion).
		Updates(map[string]any{
			"alias":          e.Alias,
			"estado":         e.Estado,
			"cerrado_en":     e.CerradoEn,
			"version":        gorm.Expr("version + 1"),
			"actualizado_en": time.Now(),
		})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrVersionMismatch
	}
	return nil
}

// ---------------- A24 · expedientes.episodios ----------------

type episodioRepo struct{ db *gorm.DB }

func (r *episodioRepo) Create(ctx context.Context, ep *models.Episodio) error {
	return translate(r.db.WithContext(ctx).Create(ep).Error)
}

func (r *episodioRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Episodio, error) {
	return first[models.Episodio](ctx, r.db, "id = ?", id)
}

func (r *episodioRepo) GetActivo(ctx context.Context, expedienteID uuid.UUID) (*models.Episodio, error) {
	return first[models.Episodio](ctx, r.db, "expediente_id = ? AND cerrado_en IS NULL", expedienteID)
}

func (r *episodioRepo) ListByExpediente(ctx context.Context, expedienteID uuid.UUID) ([]models.Episodio, error) {
	return findAll[models.Episodio](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.Where("expediente_id = ?", expedienteID).Order("numero ASC")
	})
}

func (r *episodioRepo) SiguienteNumero(ctx context.Context, expedienteID uuid.UUID) (int, error) {
	var n int
	err := r.db.WithContext(ctx).Raw(
		"SELECT COALESCE(MAX(numero), 0) + 1 FROM expedientes.episodios WHERE expediente_id = ?",
		expedienteID,
	).Scan(&n).Error
	return n, translate(err)
}

func (r *episodioRepo) Cerrar(ctx context.Context, id uuid.UUID, at time.Time) error {
	res := r.db.WithContext(ctx).Model(&models.Episodio{}).
		Where("id = ? AND cerrado_en IS NULL", id).
		Update("cerrado_en", at)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *episodioRepo) ActualizarTipoCaso(ctx context.Context, id, tipoCasoID uuid.UUID) error {
	res := r.db.WithContext(ctx).Model(&models.Episodio{}).
		Where("id = ?", id).Update("tipo_caso_id", tipoCasoID)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// ---------------- A25 · expedientes.historial_estados ----------------

type historialEstadoRepo struct{ db *gorm.DB }

func (r *historialEstadoRepo) Append(ctx context.Context, h *models.HistorialEstado) error {
	return translate(r.db.WithContext(ctx).Create(h).Error)
}

func (r *historialEstadoRepo) ListByExpediente(ctx context.Context, expedienteID uuid.UUID, page repository.PageRequest) (repository.Page[models.HistorialEstado], error) {
	q := r.db.WithContext(ctx).Model(&models.HistorialEstado{}).
		Where("expediente_id = ?", expedienteID)

	limit := pageLimit(page)
	q = applyPage(q, page, "creado_en", "id", true)

	var rows []models.HistorialEstado
	if err := q.Find(&rows).Error; err != nil {
		return repository.Page[models.HistorialEstado]{}, translate(err)
	}
	rows, next := cutPage(rows, limit,
		func(m models.HistorialEstado) time.Time { return m.CreadoEn },
		func(m models.HistorialEstado) uuid.UUID { return m.ID },
	)
	return repository.Page[models.HistorialEstado]{Items: rows, NextCursor: next}, nil
}

// ---------------- A26 · expedientes.asignaciones ----------------

type asignacionRepo struct{ db *gorm.DB }

func (r *asignacionRepo) Create(ctx context.Context, a *models.Asignacion) error {
	return translate(r.db.WithContext(ctx).Create(a).Error)
}

func (r *asignacionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Asignacion, error) {
	return first[models.Asignacion](ctx, r.db, "id = ?", id)
}

func (r *asignacionRepo) FindVigente(ctx context.Context, expedienteID, personalID, departamentoID uuid.UUID) (*models.Asignacion, error) {
	return first[models.Asignacion](ctx, r.db,
		"expediente_id = ? AND personal_id = ? AND departamento_id = ? AND fin IS NULL",
		expedienteID, personalID, departamentoID,
	)
}

func (r *asignacionRepo) ListByExpediente(ctx context.Context, expedienteID uuid.UUID, soloVigentes bool) ([]models.Asignacion, error) {
	return findAll[models.Asignacion](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		q = q.Where("expediente_id = ?", expedienteID)
		if soloVigentes {
			q = q.Where("fin IS NULL")
		}
		return q.Order("inicio ASC")
	})
}

func (r *asignacionRepo) ListVigentesByPersonal(ctx context.Context, personalID uuid.UUID) ([]models.Asignacion, error) {
	return findAll[models.Asignacion](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.Where("personal_id = ? AND fin IS NULL", personalID).Order("inicio ASC")
	})
}

func (r *asignacionRepo) TieneAcceso(ctx context.Context, personalID, expedienteID uuid.UUID, departamentoID *uuid.UUID) (bool, error) {
	q := r.db.WithContext(ctx).Model(&models.Asignacion{}).
		Where("personal_id = ? AND expediente_id = ? AND fin IS NULL", personalID, expedienteID)
	if departamentoID != nil {
		q = q.Where("departamento_id = ?", *departamentoID)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		return false, translate(err)
	}
	return n > 0, nil
}

func (r *asignacionRepo) DepartamentosAsignados(ctx context.Context, personalID, expedienteID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).Model(&models.Asignacion{}).
		Where("personal_id = ? AND expediente_id = ? AND fin IS NULL", personalID, expedienteID).
		Distinct().Pluck("departamento_id", &ids).Error
	return ids, translate(err)
}

func (r *asignacionRepo) Cerrar(ctx context.Context, id uuid.UUID, fin time.Time) error {
	res := r.db.WithContext(ctx).Model(&models.Asignacion{}).
		Where("id = ? AND fin IS NULL", id).
		Update("fin", fin)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// ---------------- A27 · expedientes.notas ----------------

type notaRepo struct{ db *gorm.DB }

func (r *notaRepo) Create(ctx context.Context, n *models.Nota) error {
	return translate(r.db.WithContext(ctx).Create(n).Error)
}

func (r *notaRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Nota, error) {
	return first[models.Nota](ctx, r.db, "id = ?", id)
}

func (r *notaRepo) GetByOperacionID(ctx context.Context, operationID uuid.UUID) (*models.Nota, error) {
	return first[models.Nota](ctx, r.db, "operacion_id = ?", operationID)
}

func (r *notaRepo) ListBySeccion(ctx context.Context, expedienteID, departamentoID uuid.UUID, f repository.NotaFilter, page repository.PageRequest) (repository.Page[models.Nota], error) {
	q := r.db.WithContext(ctx).Model(&models.Nota{}).
		Where("expediente_id = ? AND departamento_id = ?", expedienteID, departamentoID)
	if f.EpisodioID != nil {
		q = q.Where("episodio_id = ?", *f.EpisodioID)
	}
	q = applyTimeRange(q, f.Rango, "creado_en")

	limit := pageLimit(page)
	q = applyPage(q, page, "creado_en", "id", true)

	var rows []models.Nota
	if err := q.Find(&rows).Error; err != nil {
		return repository.Page[models.Nota]{}, translate(err)
	}
	rows, next := cutPage(rows, limit,
		func(m models.Nota) time.Time { return m.CreadoEn },
		func(m models.Nota) uuid.UUID { return m.ID },
	)
	return repository.Page[models.Nota]{Items: rows, NextCursor: next}, nil
}

func (r *notaRepo) EsPrevia(ctx context.Context, corrigeNotaID, expedienteID, departamentoID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.Nota{}).
		Where("id = ? AND expediente_id = ? AND departamento_id = ?", corrigeNotaID, expedienteID, departamentoID).
		Count(&n).Error
	if err != nil {
		return false, translate(err)
	}
	return n > 0, nil
}

func (r *notaRepo) TieneCorreccion(ctx context.Context, notaID uuid.UUID) (bool, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&models.Nota{}).Where("corrige_nota_id = ?", notaID).Count(&n).Error; err != nil {
		return false, translate(err)
	}
	return n > 0, nil
}

// ---------------- A23 · expedientes.v_timeline_autorizado ----------------

type timelineRepo struct{ db *gorm.DB }

func (r *timelineRepo) ListAutorizado(ctx context.Context, expedienteID uuid.UUID, f repository.TimelineFilter, page repository.PageRequest) (repository.Page[models.TimelineItem], error) {
	q := r.db.WithContext(ctx).Model(&models.TimelineItem{}).
		Where("expediente_id = ?", expedienteID)
	if len(f.DepartamentoIDs) > 0 {
		q = q.Where("departamento_id IN ?", f.DepartamentoIDs)
	}
	if len(f.Tipos) > 0 {
		q = q.Where("tipo IN ?", f.Tipos)
	}
	q = applyTimeRange(q, f.Rango, "ocurrido_en")

	limit := pageLimit(page)
	q = applyPage(q, page, "ocurrido_en", "item_id", true)

	var rows []models.TimelineItem
	if err := q.Find(&rows).Error; err != nil {
		return repository.Page[models.TimelineItem]{}, translate(err)
	}
	rows, next := cutPage(rows, limit,
		func(m models.TimelineItem) time.Time { return m.OcurridoEn },
		func(m models.TimelineItem) uuid.UUID { return m.ItemID },
	)
	return repository.Page[models.TimelineItem]{Items: rows, NextCursor: next}, nil
}

// ---------------- A28 · expedientes.tokens_vinculacion ----------------

type tokenVinculacionRepo struct{ db *gorm.DB }

func (r *tokenVinculacionRepo) Create(ctx context.Context, t *models.TokenVinculacion) error {
	return translate(r.db.WithContext(ctx).Create(t).Error)
}

func (r *tokenVinculacionRepo) GetByTokenHash(ctx context.Context, tokenHash []byte) (*models.TokenVinculacion, error) {
	return first[models.TokenVinculacion](ctx, r.db, "token_hash = ?", tokenHash)
}

func (r *tokenVinculacionRepo) Consumir(ctx context.Context, id uuid.UUID, at time.Time) (bool, error) {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE expedientes.tokens_vinculacion
		SET consumido_en = ?
		WHERE id = ? AND consumido_en IS NULL AND vence_en > ?`,
		at, id, at)
	if res.Error != nil {
		return false, translate(res.Error)
	}
	return res.RowsAffected > 0, nil
}

func (r *tokenVinculacionRepo) PurgarVencidos(ctx context.Context, before time.Time) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("vence_en < ?", before).
		Delete(&models.TokenVinculacion{})
	return res.RowsAffected, translate(res.Error)
}

// ---------------- A29 · expedientes.solicitudes_vinculacion ----------------

type solicitudVinculacionRepo struct{ db *gorm.DB }

func (r *solicitudVinculacionRepo) Create(ctx context.Context, s *models.SolicitudVinculacion) error {
	return translate(r.db.WithContext(ctx).Create(s).Error)
}

func (r *solicitudVinculacionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.SolicitudVinculacion, error) {
	return first[models.SolicitudVinculacion](ctx, r.db, "id = ?", id)
}

func (r *solicitudVinculacionRepo) GetByTokenID(ctx context.Context, tokenID uuid.UUID) (*models.SolicitudVinculacion, error) {
	return first[models.SolicitudVinculacion](ctx, r.db, "token_id = ?", tokenID)
}

func (r *solicitudVinculacionRepo) List(ctx context.Context, f repository.SolicitudVinculacionFilter, page repository.PageRequest) (repository.Page[models.SolicitudVinculacion], error) {
	q := r.db.WithContext(ctx).Model(&models.SolicitudVinculacion{})
	if f.Estado != nil {
		q = q.Where("estado = ?", *f.Estado)
	}
	if f.SolicitanteID != nil {
		q = q.Where("solicitante_id = ?", *f.SolicitanteID)
	}

	limit := pageLimit(page)
	q = applyPage(q, page, "creado_en", "id", false)

	var rows []models.SolicitudVinculacion
	if err := q.Find(&rows).Error; err != nil {
		return repository.Page[models.SolicitudVinculacion]{}, translate(err)
	}
	rows, next := cutPage(rows, limit,
		func(m models.SolicitudVinculacion) time.Time { return m.CreadoEn },
		func(m models.SolicitudVinculacion) uuid.UUID { return m.ID },
	)
	return repository.Page[models.SolicitudVinculacion]{Items: rows, NextCursor: next}, nil
}

func (r *solicitudVinculacionRepo) Resolver(ctx context.Context, id uuid.UUID, estado models.EstadoSolicitudVinculacion, resueltaPor uuid.UUID, resueltaEn time.Time, asignacionID *uuid.UUID, expectedVersion int64) error {
	res := r.db.WithContext(ctx).Model(&models.SolicitudVinculacion{}).
		Where("id = ? AND version = ?", id, expectedVersion).
		Updates(map[string]any{
			"estado":         estado,
			"resuelta_por":   resueltaPor,
			"resuelta_en":    resueltaEn,
			"asignacion_id":  asignacionID,
			"version":        gorm.Expr("version + 1"),
			"actualizado_en": time.Now(),
		})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrVersionMismatch
	}
	return nil
}
