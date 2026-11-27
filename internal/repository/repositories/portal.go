package repository

import (
	"context"
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

type PortalCuentaRepository interface {
	Create(ctx context.Context, c *models.PortalCuenta) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.PortalCuenta, error)
	GetByCURPNormalizada(ctx context.Context, curp string) (*models.PortalCuenta, error)
	FindByNombre(ctx context.Context, nombre string) (*models.PortalCuenta, error)
	Update(ctx context.Context, c *models.PortalCuenta) error
}

type PortalContactoRepository interface {
	ListByCuentaID(ctx context.Context, cuentaID uuid.UUID) ([]models.PortalContactoConfianza, error)
}

type PortalDesafioRepository interface {
	Create(ctx context.Context, d *models.PortalDesafioOTP) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.PortalDesafioOTP, error)
	Save(ctx context.Context, d *models.PortalDesafioOTP) error
}

type PortalSesionRepository interface {
	Create(ctx context.Context, s *models.PortalSesion) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.PortalSesion, error)
	Touch(ctx context.Context, id uuid.UUID, at time.Time) error
	Revoke(ctx context.Context, id uuid.UUID, at time.Time) error
}

type PortalAlertaRepository interface {
	Create(ctx context.Context, a *models.PortalAlertaAyuda) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.PortalAlertaAyuda, error)
	GetByOperacionID(ctx context.Context, cuentaID, opID uuid.UUID) (*models.PortalAlertaAyuda, error)
	Update(ctx context.Context, a *models.PortalAlertaAyuda) error
}
