package repository

import (
	"context"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

type VacanteFilter struct {
	Sector      string
	Horario     string
	TipoEmpleo  string
	Ubicacion   string
	EmpleadorID *uuid.UUID
	Estado      *models.EstadoVacante
}

type VacanteRepository interface {
	Create(ctx context.Context, v *models.Vacante) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Vacante, error)
	Save(ctx context.Context, v *models.Vacante) error
	List(ctx context.Context, f VacanteFilter, page PageRequest) (Page[models.Vacante], error)
	CountPostulaciones(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]int, error)
}

type PostulacionRepository interface {
	Create(ctx context.Context, p *models.Postulacion) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Postulacion, error)
	Save(ctx context.Context, p *models.Postulacion) error
	Exists(ctx context.Context, vacanteID, usuariaID uuid.UUID) (bool, error)
	ExistsMany(ctx context.Context, vacanteIDs []uuid.UUID, usuariaID uuid.UUID) (map[uuid.UUID]bool, error)
	ListByUsuaria(ctx context.Context, usuariaID uuid.UUID) ([]models.Postulacion, error)
	ListByVacante(ctx context.Context, vacanteID uuid.UUID) ([]models.Postulacion, error)
}
