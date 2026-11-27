package repository

import (
	"context"
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

type CursoFilter struct {
	Tipo         *models.TipoCurso
	Desde, Hasta *time.Time
}

type CursoRepository interface {
	Create(ctx context.Context, c *models.Curso) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Curso, error)
	List(ctx context.Context, f CursoFilter, page PageRequest) (Page[models.Curso], error)
	Inscritas(ctx context.Context, cursoIDs []uuid.UUID) (map[uuid.UUID]int, error)
}

type InscripcionRepository interface {
	Create(ctx context.Context, i *models.Inscripcion) error
	GetByCursoAndUsuaria(ctx context.Context, cursoID, usuariaID uuid.UUID) (*models.Inscripcion, error)
	ExistsMany(ctx context.Context, cursoIDs []uuid.UUID, usuariaID uuid.UUID) (map[uuid.UUID]bool, error)
	ListByCurso(ctx context.Context, cursoID uuid.UUID) ([]models.Inscripcion, error)
	SetPresente(ctx context.Context, cursoID, inscripcionID uuid.UUID, presente bool) error
}

type VisitaRepository interface {
	Create(ctx context.Context, v *models.Visita) error
	List(ctx context.Context, estado *models.EstadoVisita, page PageRequest) (Page[models.Visita], error)
}
