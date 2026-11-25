package gormrepo

import (
	"context"
	"time"

	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---------------- A01 · catalogos.departamentos ----------------

type departamentoRepo struct{ db *gorm.DB }

func (r *departamentoRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Departamento, error) {
	return first[models.Departamento](ctx, r.db, "id = ?", id)
}

func (r *departamentoRepo) GetByCodigo(ctx context.Context, codigo string) (*models.Departamento, error) {
	return first[models.Departamento](ctx, r.db, "codigo = ?", codigo)
}

func (r *departamentoRepo) List(ctx context.Context, soloActivos bool) ([]models.Departamento, error) {
	return findAll[models.Departamento](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		if soloActivos {
			q = q.Where("activo = ?", true)
		}
		return q.Order("nombre ASC")
	})
}

// ---------------- A02 · catalogos.roles ----------------

type rolRepo struct{ db *gorm.DB }

func (r *rolRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Rol, error) {
	return first[models.Rol](ctx, r.db, "id = ?", id)
}

func (r *rolRepo) GetByCodigo(ctx context.Context, codigo string) (*models.Rol, error) {
	return first[models.Rol](ctx, r.db, "codigo = ?", codigo)
}

func (r *rolRepo) List(ctx context.Context, soloActivos bool) ([]models.Rol, error) {
	return findAll[models.Rol](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		if soloActivos {
			q = q.Where("activo = ?", true)
		}
		return q.Order("nombre ASC")
	})
}

// ---------------- A03 · catalogos.permisos ----------------

type permisoRepo struct{ db *gorm.DB }

func (r *permisoRepo) GetByCodigo(ctx context.Context, codigo string) (*models.Permiso, error) {
	return first[models.Permiso](ctx, r.db, "codigo = ?", codigo)
}

func (r *permisoRepo) List(ctx context.Context) ([]models.Permiso, error) {
	return findAll[models.Permiso](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.Order("codigo ASC")
	})
}

func (r *permisoRepo) ListByRol(ctx context.Context, rolID uuid.UUID) ([]models.Permiso, error) {
	return findAll[models.Permiso](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.
			Joins("JOIN catalogos.rol_permisos rp ON rp.permiso_id = catalogos.permisos.id").
			Where("rp.rol_id = ?", rolID).
			Order("catalogos.permisos.codigo ASC")
	})
}

func (r *permisoRepo) CodigosEfectivos(ctx context.Context, personalID uuid.UUID, at time.Time) ([]string, error) {
	var codigos []string
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT p.codigo
		FROM identidad.personal pe
		JOIN catalogos.rol_permisos rp ON rp.rol_id = pe.rol_id
		JOIN catalogos.permisos p ON p.id = rp.permiso_id
		WHERE pe.id = ?
		UNION
		SELECT DISTINCT p.codigo
		FROM identidad.delegaciones d
		JOIN catalogos.permisos p ON p.id = d.permiso_id
		WHERE d.personal_id = ?
		  AND d.revocado_en IS NULL
		  AND d.inicio <= ?
		  AND d.fin > ?`,
		personalID, personalID, at, at).Scan(&codigos).Error
	return codigos, translate(err)
}

// ---------------- A04 · catalogos.rol_permisos ----------------

type rolPermisoRepo struct{ db *gorm.DB }

func (r *rolPermisoRepo) Add(ctx context.Context, rolID, permisoID uuid.UUID) (*models.RolPermiso, error) {
	rp := &models.RolPermiso{RolID: rolID, PermisoID: permisoID}
	if err := r.db.WithContext(ctx).Create(rp).Error; err != nil {
		return nil, translate(err)
	}
	return rp, nil
}

func (r *rolPermisoRepo) Remove(ctx context.Context, rolID, permisoID uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("rol_id = ? AND permiso_id = ?", rolID, permisoID).
		Delete(&models.RolPermiso{})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *rolPermisoRepo) ListByRol(ctx context.Context, rolID uuid.UUID) ([]models.RolPermiso, error) {
	return findAll[models.RolPermiso](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.Where("rol_id = ?", rolID).Order("creado_en ASC")
	})
}

// ---------------- A05 · catalogos.sectores ----------------

type sectorRepo struct{ db *gorm.DB }

func (r *sectorRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Sector, error) {
	return first[models.Sector](ctx, r.db, "id = ?", id)
}

func (r *sectorRepo) GetByCodigo(ctx context.Context, codigo string) (*models.Sector, error) {
	return first[models.Sector](ctx, r.db, "codigo = ?", codigo)
}

func (r *sectorRepo) List(ctx context.Context, soloActivos bool) ([]models.Sector, error) {
	return findAll[models.Sector](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		if soloActivos {
			q = q.Where("activo = ?", true)
		}
		return q.Order("nombre ASC")
	})
}

// ---------------- A06 · catalogos.tipos_caso ----------------

type tipoCasoRepo struct{ db *gorm.DB }

func (r *tipoCasoRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.TipoCaso, error) {
	return first[models.TipoCaso](ctx, r.db, "id = ?", id)
}

func (r *tipoCasoRepo) GetByCodigo(ctx context.Context, codigo string) (*models.TipoCaso, error) {
	return first[models.TipoCaso](ctx, r.db, "codigo = ?", codigo)
}

func (r *tipoCasoRepo) List(ctx context.Context, soloActivos bool) ([]models.TipoCaso, error) {
	return findAll[models.TipoCaso](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		if soloActivos {
			q = q.Where("activo = ?", true)
		}
		return q.Order("nombre ASC")
	})
}

// ---------------- A07 · catalogos.instituciones ----------------

type institucionRepo struct{ db *gorm.DB }

func (r *institucionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Institucion, error) {
	return first[models.Institucion](ctx, r.db, "id = ?", id)
}

func (r *institucionRepo) List(ctx context.Context, soloActivas bool) ([]models.Institucion, error) {
	return findAll[models.Institucion](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		if soloActivas {
			q = q.Where("activo = ?", true)
		}
		return q.Order("nombre ASC")
	})
}

func (r *institucionRepo) ListPublicables(ctx context.Context) ([]models.Institucion, error) {
	return findAll[models.Institucion](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.Where("publicable = ? AND activo = ?", true, true).Order("nombre ASC")
	})
}

func (r *institucionRepo) Create(ctx context.Context, inst *models.Institucion) error {
	return translate(r.db.WithContext(ctx).Create(inst).Error)
}

func (r *institucionRepo) Update(ctx context.Context, inst *models.Institucion, expectedVersion int64) error {
	res := r.db.WithContext(ctx).Model(&models.Institucion{}).
		Where("id = ? AND version = ?", inst.ID, expectedVersion).
		Updates(map[string]any{
			"nombre":         inst.Nombre,
			"telefono":       inst.Telefono,
			"horario":        inst.Horario,
			"ruta_atencion":  inst.RutaAtencion,
			"publicable":     inst.Publicable,
			"verificado_en":  inst.VerificadoEn,
			"activo":         inst.Activo,
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

// ---------------- A08 · catalogos.propositos ----------------

type propositoRepo struct{ db *gorm.DB }

func (r *propositoRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Proposito, error) {
	return first[models.Proposito](ctx, r.db, "id = ?", id)
}

func (r *propositoRepo) GetByCodigo(ctx context.Context, codigo string) (*models.Proposito, error) {
	return first[models.Proposito](ctx, r.db, "codigo = ?", codigo)
}

func (r *propositoRepo) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]models.Proposito, error) {
	if len(ids) == 0 {
		return []models.Proposito{}, nil
	}
	return findAll[models.Proposito](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.Where("id IN ?", ids).Order("codigo ASC")
	})
}

func (r *propositoRepo) List(ctx context.Context, soloActivos bool) ([]models.Proposito, error) {
	return findAll[models.Proposito](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		if soloActivos {
			q = q.Where("activo = ?", true)
		}
		return q.Order("codigo ASC")
	})
}

func (r *propositoRepo) ListByAviso(ctx context.Context, avisoID uuid.UUID) ([]models.Proposito, error) {
	return findAll[models.Proposito](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.
			Joins("JOIN catalogos.aviso_propositos ap ON ap.proposito_id = catalogos.propositos.id").
			Where("ap.aviso_id = ?", avisoID).
			Order("catalogos.propositos.codigo ASC")
	})
}

// ---------------- A09 · catalogos.avisos_privacidad ----------------

type avisoPrivacidadRepo struct{ db *gorm.DB }

func (r *avisoPrivacidadRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.AvisoPrivacidad, error) {
	return first[models.AvisoPrivacidad](ctx, r.db, "id = ?", id)
}

func (r *avisoPrivacidadRepo) GetByCodigoVersion(ctx context.Context, codigoVersion string) (*models.AvisoPrivacidad, error) {
	return first[models.AvisoPrivacidad](ctx, r.db, "codigo_version = ?", codigoVersion)
}

func (r *avisoPrivacidadRepo) GetVigente(ctx context.Context, at time.Time) (*models.AvisoPrivacidad, error) {
	var out models.AvisoPrivacidad
	err := r.db.WithContext(ctx).
		Where("publicado_en <= ? AND retirado_en IS NULL", at).
		Order("publicado_en DESC").
		First(&out).Error
	if err != nil {
		return nil, translate(err)
	}
	return &out, nil
}

func (r *avisoPrivacidadRepo) Create(ctx context.Context, aviso *models.AvisoPrivacidad) error {
	return translate(r.db.WithContext(ctx).Create(aviso).Error)
}

func (r *avisoPrivacidadRepo) Retirar(ctx context.Context, id uuid.UUID, at time.Time) error {
	res := r.db.WithContext(ctx).Model(&models.AvisoPrivacidad{}).
		Where("id = ? AND retirado_en IS NULL", id).
		Update("retirado_en", at)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// ---------------- A10 · catalogos.aviso_propositos ----------------

type avisoPropositoRepo struct{ db *gorm.DB }

func (r *avisoPropositoRepo) Add(ctx context.Context, avisoID, propositoID uuid.UUID) (*models.AvisoProposito, error) {
	ap := &models.AvisoProposito{AvisoID: avisoID, PropositoID: propositoID}
	if err := r.db.WithContext(ctx).Create(ap).Error; err != nil {
		return nil, translate(err)
	}
	return ap, nil
}

func (r *avisoPropositoRepo) ListByAviso(ctx context.Context, avisoID uuid.UUID) ([]models.AvisoProposito, error) {
	return findAll[models.AvisoProposito](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.Where("aviso_id = ?", avisoID).Order("creado_en ASC")
	})
}

func (r *avisoPropositoRepo) AdmiteTodos(ctx context.Context, avisoID uuid.UUID, propositoIDs []uuid.UUID) (bool, error) {
	uniq := make(map[uuid.UUID]struct{}, len(propositoIDs))
	for _, id := range propositoIDs {
		uniq[id] = struct{}{}
	}
	if len(uniq) == 0 {
		return true, nil
	}
	keys := make([]uuid.UUID, 0, len(uniq))
	for id := range uniq {
		keys = append(keys, id)
	}
	var n int64
	err := r.db.WithContext(ctx).Model(&models.AvisoProposito{}).
		Where("aviso_id = ? AND proposito_id IN ?", avisoID, keys).
		Count(&n).Error
	if err != nil {
		return false, translate(err)
	}
	return n == int64(len(uniq)), nil
}
