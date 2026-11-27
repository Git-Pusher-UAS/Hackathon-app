package repository

import (
	"context"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

type VacanteFilter struct {
	Sector     string
	Horario    string
	TipoEmpleo string
	Ubicacion  string
}

type VacanteRepository interface {
	Create(ctx context.Context, v *models.Vacante) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Vacante, error)
	List(ctx context.Context, f VacanteFilter, page PageRequest) (Page[models.Vacante], error)
	CountPostulaciones(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]int, error)
}

type PostulacionRepository interface {
	Create(ctx context.Context, p *models.Postulacion) error
	Exists(ctx context.Context, vacanteID, usuariaID uuid.UUID) (bool, error)
	ExistsMany(ctx context.Context, vacanteIDs []uuid.UUID, usuariaID uuid.UUID) (map[uuid.UUID]bool, error)
}
