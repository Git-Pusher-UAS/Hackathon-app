// Package audit centraliza la escritura de eventos de auditoría.
package audit

import (
	"context"
	"time"

	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

// Record inserta un evento de auditoría sobre el tipo de recurso "personal".
// Debe llamarse dentro de la misma transacción que la escritura de negocio
// cuando se quiera confirmar o revertir ambos juntos.
func Record(
	ctx context.Context,
	repos repository.Repositories,
	actorID *uuid.UUID,
	accion string,
	resultado models.ResultadoAuditoria,
	recursoID *uuid.UUID,
	requestID uuid.UUID,
	ip *string,
) error {
	return RecordResource(ctx, repos, actorID, accion, "personal", resultado, recursoID, requestID, ip)
}

// RecordResource inserta un evento de auditoría indicando el tipo de recurso afectado.
func RecordResource(
	ctx context.Context,
	repos repository.Repositories,
	actorID *uuid.UUID,
	accion string,
	tipoRecurso string,
	resultado models.ResultadoAuditoria,
	recursoID *uuid.UUID,
	requestID uuid.UUID,
	ip *string,
) error {
	if requestID == uuid.Nil {
		requestID = uuid.New()
	}
	return repos.EventosAuditoria().Append(ctx, &models.EventoAuditoria{
		ActorID:     actorID,
		Accion:      accion,
		TipoRecurso: tipoRecurso,
		RecursoID:   recursoID,
		Resultado:   resultado,
		OcurridoEn:  time.Now(),
		IPOrigen:    ip,
		RequestID:   requestID,
	})
}

func RecordWithMeta(ctx context.Context, repos repository.Repositories, actorID *uuid.UUID, accion, tipoRecurso string, resultado models.ResultadoAuditoria, recursoID *uuid.UUID, requestID uuid.UUID, ip *string, meta []byte) error {
	if requestID == uuid.Nil {
		requestID = uuid.New()
	}
	return repos.EventosAuditoria().Append(ctx, &models.EventoAuditoria{ActorID: actorID, Accion: accion, TipoRecurso: tipoRecurso, RecursoID: recursoID, Resultado: resultado, Metadatos: meta, OcurridoEn: time.Now(), IPOrigen: ip, RequestID: requestID})
}
