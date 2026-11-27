package repository

import (
	"backend-hackathon/internal/models"
	"context"
	"github.com/google/uuid"
	"time"
)

// AuditoriaFilter para API-035.
type AuditoriaFilter struct {
	ActorID     *uuid.UUID
	Accion      string
	TipoRecurso string
	RecursoID   *uuid.UUID
	Resultado   *models.ResultadoAuditoria
	Rango       TimeRange // sobre ocurrido_en
}

// A36 · auditoria.eventos — append-only, encadenado por hash.
type EventoAuditoriaRepository interface {
	// Append asigna secuencia, hash_anterior y hash_actual bajo un candado (p. ej.
	// pg_advisory_xact_lock) que serializa el encadenamiento, e inserta el evento.
	// El llamador rellena los campos de negocio; la implementación completa los de cadena.
	Append(ctx context.Context, e *models.EventoAuditoria) error

	// Ultimo devuelve el último evento de la cadena (con candado), útil para calcular el hash.
	Ultimo(ctx context.Context) (*models.EventoAuditoria, error)

	GetByID(ctx context.Context, id uuid.UUID) (*models.EventoAuditoria, error)
	List(ctx context.Context, f AuditoriaFilter, page PageRequest) (Page[models.EventoAuditoria], error)

	// Contar cuenta eventos de un actor para una acción en un rango (detección de alertas).
	Contar(ctx context.Context, actorID uuid.UUID, acciones []string, rango TimeRange) (int, error)

	// RangoSecuencia devuelve eventos por secuencia (desde, hasta], para verificar la cadena.
	RangoSecuencia(ctx context.Context, desdeExclusivo, hastaInclusivo int64) ([]models.EventoAuditoria, error)
}

// AlertaFilter para API-036.
type AlertaFilter struct {
	ActorObservadoID *uuid.UUID
	Nivel            *models.NivelAlerta
	Estado           *models.EstadoAlerta
	Rango            TimeRange // sobre creado_en
}

// A37 · auditoria.alertas
type AlertaAuditoriaRepository interface {
	// CreateIfAbsent inserta la alerta si no existe (actor, tipo, ventana). Devuelve
	// created=false si ya existía (evita alertas duplicadas).
	CreateIfAbsent(ctx context.Context, a *models.AlertaAuditoria) (created bool, err error)

	GetByID(ctx context.Context, id uuid.UUID) (*models.AlertaAuditoria, error)
	List(ctx context.Context, f AlertaFilter, page PageRequest) (Page[models.AlertaAuditoria], error)

	// MarcarRevisada cambia estado abierta → revisada sin alterar el evento original.
	MarcarRevisada(ctx context.Context, id uuid.UUID) error
}

// A38 · ayuda.eventos_locales
type EventoLocalAyudaRepository interface {
	// Create devuelve ErrConflict si operacion_id ya existe (reintento).
	Create(ctx context.Context, e *models.EventoLocalAyuda) error
	GetByOperacionID(ctx context.Context, operacionID uuid.UUID) (*models.EventoLocalAyuda, error)
	ListByPersonal(ctx context.Context, personalID uuid.UUID, page PageRequest) (Page[models.EventoLocalAyuda], error)
}

// ResumenFiltro limita las dimensiones aprobadas del resumen (API-037).
type ResumenFiltro struct {
	SectorID   *uuid.UUID
	TipoCasoID *uuid.UUID
	Rango      TimeRange
}

// analitica.v_resumen_mvp. Aplica k = 5; una celda suprimida llega como nil.
type ResumenRepository interface {
	Consultar(ctx context.Context, f ResumenFiltro) ([]models.ResumenFila, error)
}

// MapaFiltro limita el mapa (API-038).
type MapaFiltro struct {
	TipoCasoID *uuid.UUID
	Periodo    *time.Time
}

// analitica.v_mapa_mvp. Sin puntos domiciliarios; celdas pequeñas suprimidas.
type MapaRepository interface {
	Consultar(ctx context.Context, f MapaFiltro) ([]models.MapaCelda, error)
}

// Conteo es una cifra agregada por etiqueta.
type Conteo struct {
	Etiqueta string
	Valor    int
}

// IndicadoresFiltro limita agregados del tablero.
type IndicadoresFiltro struct {
	Rango      TimeRange
	TipoCasoID *uuid.UUID
	Estado     *models.EstadoExpediente
}

// IndicadoresRepository calcula agregados del tablero.
type IndicadoresRepository interface {
	Expedientes(ctx context.Context, f IndicadoresFiltro) (int, error)
	Canalizaciones(ctx context.Context, f IndicadoresFiltro) (int, error)
	PorMes(ctx context.Context, f IndicadoresFiltro) ([]Conteo, error)
	PorOrigen(ctx context.Context, f IndicadoresFiltro) ([]Conteo, error)
	PorArea(ctx context.Context, f IndicadoresFiltro) ([]Conteo, error)
	PorSector(ctx context.Context, f IndicadoresFiltro) ([]Conteo, error)
}
