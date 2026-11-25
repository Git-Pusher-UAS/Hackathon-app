package repository

import (
	"backend-hackathon/internal/models"
	"context"
	"time"

	"github.com/google/uuid"
)

// A01 · catalogos.departamentos
type DepartamentoRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Departamento, error)
	GetByCodigo(ctx context.Context, codigo string) (*models.Departamento, error)
	List(ctx context.Context, soloActivos bool) ([]models.Departamento, error)
}

// A02 · catalogos.roles
type RolRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Rol, error)
	GetByCodigo(ctx context.Context, codigo string) (*models.Rol, error)
	List(ctx context.Context, soloActivos bool) ([]models.Rol, error)
}

// A03 · catalogos.permisos
type PermisoRepository interface {
	GetByCodigo(ctx context.Context, codigo string) (*models.Permiso, error)
	List(ctx context.Context) ([]models.Permiso, error)
	ListByRol(ctx context.Context, rolID uuid.UUID) ([]models.Permiso, error)

	// CodigosEfectivos devuelve los códigos de permiso del rol principal del personal
	// más sus delegaciones vigentes (no revocadas y con inicio <= at < fin).
	// No existe permiso universal implícito.
	CodigosEfectivos(ctx context.Context, personalID uuid.UUID, at time.Time) ([]string, error)
}

// A04 · catalogos.rol_permisos
type RolPermisoRepository interface {
	Add(ctx context.Context, rolID, permisoID uuid.UUID) (*models.RolPermiso, error)
	Remove(ctx context.Context, rolID, permisoID uuid.UUID) error
	ListByRol(ctx context.Context, rolID uuid.UUID) ([]models.RolPermiso, error)
}

// A05 · catalogos.sectores
type SectorRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Sector, error)
	GetByCodigo(ctx context.Context, codigo string) (*models.Sector, error)
	List(ctx context.Context, soloActivos bool) ([]models.Sector, error)
}

// A06 · catalogos.tipos_caso
type TipoCasoRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.TipoCaso, error)
	GetByCodigo(ctx context.Context, codigo string) (*models.TipoCaso, error)
	List(ctx context.Context, soloActivos bool) ([]models.TipoCaso, error)
}

// A07 · catalogos.instituciones
type InstitucionRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Institucion, error)
	List(ctx context.Context, soloActivas bool) ([]models.Institucion, error)

	// ListPublicables sirve GET /api/v1/directory: publicable = true y activo = true.
	ListPublicables(ctx context.Context) ([]models.Institucion, error)

	Create(ctx context.Context, inst *models.Institucion) error

	// Update guarda cambios si la versión coincide; incrementa version y actualizado_en.
	Update(ctx context.Context, inst *models.Institucion, expectedVersion int64) error
}

// A08 · catalogos.propositos
type PropositoRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Proposito, error)
	GetByCodigo(ctx context.Context, codigo string) (*models.Proposito, error)
	ListByIDs(ctx context.Context, ids []uuid.UUID) ([]models.Proposito, error)
	List(ctx context.Context, soloActivos bool) ([]models.Proposito, error)

	// ListByAviso devuelve las finalidades admitidas por una versión de aviso.
	ListByAviso(ctx context.Context, avisoID uuid.UUID) ([]models.Proposito, error)
}

// A09 · catalogos.avisos_privacidad (versiones inmutables)
type AvisoPrivacidadRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.AvisoPrivacidad, error)
	GetByCodigoVersion(ctx context.Context, codigoVersion string) (*models.AvisoPrivacidad, error)

	// GetVigente devuelve el aviso publicado y no retirado en el instante at.
	GetVigente(ctx context.Context, at time.Time) (*models.AvisoPrivacidad, error)

	Create(ctx context.Context, aviso *models.AvisoPrivacidad) error

	// Retirar marca retirado_en; nunca elimina versiones históricas.
	Retirar(ctx context.Context, id uuid.UUID, at time.Time) error
}

// A10 · catalogos.aviso_propositos
type AvisoPropositoRepository interface {
	Add(ctx context.Context, avisoID, propositoID uuid.UUID) (*models.AvisoProposito, error)
	ListByAviso(ctx context.Context, avisoID uuid.UUID) ([]models.AvisoProposito, error)

	// AdmiteTodos comprueba que todas las finalidades están incluidas en el aviso.
	AdmiteTodos(ctx context.Context, avisoID uuid.UUID, propositoIDs []uuid.UUID) (bool, error)
}
