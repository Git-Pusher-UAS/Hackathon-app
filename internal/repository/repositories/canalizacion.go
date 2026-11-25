package repository

import (
	"backend-hackathon/internal/models"
	"context"

	"github.com/google/uuid"
)

// CanalizacionFilter para las bandejas de API-032.
type CanalizacionFilter struct {
	DestinoIDs   []uuid.UUID // bandeja de entrada (departamentos receptores)
	OrigenIDs    []uuid.UUID // bandeja de salida
	EmitidaPor   *uuid.UUID
	ExpedienteID *uuid.UUID
	Estados      []models.EstadoCanalizacion
}

// A34 · canalizacion.internas
type CanalizacionRepository interface {
	// Create devuelve ErrConflict si ya hay una canalización pendiente/aceptada equivalente
	// (expediente, origen, destino) o si la operación ya fue aplicada.
	Create(ctx context.Context, c *models.CanalizacionInterna) error

	GetByID(ctx context.Context, id uuid.UUID) (*models.CanalizacionInterna, error)
	GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*models.CanalizacionInterna, error)
	GetByOperacionID(ctx context.Context, operacionID uuid.UUID) (*models.CanalizacionInterna, error)

	// List usa los índices (destino_id, estado, creado_en, id) y su análogo por origen.
	List(ctx context.Context, f CanalizacionFilter, page PageRequest) (Page[models.CanalizacionInterna], error)

	// ExisteActiva comprueba si hay canalización pendiente o aceptada (expediente, origen, destino).
	ExisteActiva(ctx context.Context, expedienteID, origenID, destinoID uuid.UUID) (bool, error)

	// Update cambia estado y asignacion_destino_id con control de versión.
	// Debe ir acompañado de HistorialCanalizacionRepository.Append en la misma transacción.
	Update(ctx context.Context, c *models.CanalizacionInterna, expectedVersion int64) error
}

// A35 · canalizacion.historial_internas (inmutable)
type HistorialCanalizacionRepository interface {
	Append(ctx context.Context, h *models.HistorialCanalizacion) error
	ListByCanalizacion(ctx context.Context, canalizacionID uuid.UUID) ([]models.HistorialCanalizacion, error)
}
