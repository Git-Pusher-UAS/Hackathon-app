package repository

import (
	"context"
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

type EmpleadorRepository interface {
	Create(ctx context.Context, e *models.Empleador) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Empleador, error)
	GetByCorreo(ctx context.Context, correo string) (*models.Empleador, error)
	List(ctx context.Context, estado *models.EstadoRegistroEmpleador) ([]models.Empleador, error)
	Save(ctx context.Context, e *models.Empleador) error
}

type EmpleadorDesafioRepository interface {
	Create(ctx context.Context, d *models.EmpleadorDesafioOTP) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.EmpleadorDesafioOTP, error)
	Save(ctx context.Context, d *models.EmpleadorDesafioOTP) error
}

type EmpleadorSesionRepository interface {
	Create(ctx context.Context, s *models.EmpleadorSesion) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.EmpleadorSesion, error)
	Touch(ctx context.Context, id uuid.UUID, at time.Time) error
	Revoke(ctx context.Context, id uuid.UUID, at time.Time) error
}

type CVRepository interface {
	GetPerfil(ctx context.Context, cuentaID uuid.UUID) (*models.CVPerfil, error)
	UpsertPerfil(ctx context.Context, p *models.CVPerfil) error
	ListExperiencias(ctx context.Context, cuentaID uuid.UUID) ([]models.CVExperiencia, error)
	ListEstudios(ctx context.Context, cuentaID uuid.UUID) ([]models.CVEstudio, error)
	ReplaceExperiencias(ctx context.Context, cuentaID uuid.UUID, items []models.CVExperiencia) error
	ReplaceEstudios(ctx context.Context, cuentaID uuid.UUID, items []models.CVEstudio) error
}
