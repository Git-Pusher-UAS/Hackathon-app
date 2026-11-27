package repository

import (
	"context"
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

// CursoFilter son los filtros de GET /courses.
type CursoFilter struct {
	Tipo         *models.TipoCurso
	Desde, Hasta *time.Time // sobre fecha, ambos inclusive
}

// CursoRepository · capacitacion.cursos
type CursoRepository interface {
	Create(ctx context.Context, c *models.Curso) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Curso, error)
	// List ordena por fecha ascendente (próximos primero).
	List(ctx context.Context, f CursoFilter, page PageRequest) (Page[models.Curso], error)
	// Inscritas cuenta las inscripciones de cada curso.
	Inscritas(ctx context.Context, cursoIDs []uuid.UUID) (map[uuid.UUID]int, error)
}

// InscripcionRepository · capacitacion.inscripciones
type InscripcionRepository interface {
	ListByCurso(ctx context.Context, cursoID uuid.UUID) ([]models.Inscripcion, error)
	// SetPresente marca la asistencia; ErrNotFound si la inscripción no es de ese curso.
	SetPresente(ctx context.Context, cursoID, inscripcionID uuid.UUID, presente bool) error
}
