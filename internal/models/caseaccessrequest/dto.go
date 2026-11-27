// Package caseaccessrequest define los DTOs de /case-access-requests (§7).
package caseaccessrequest

import (
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

// CreateRequest pide acceso con el linkToken de POST /case-files/duplicate-checks.
type CreateRequest struct {
	LinkToken string `json:"linkToken" example:"lt_9f..."`
	Motivo    string `json:"motivo" example:"La persona acudió hoy y pide atención psicológica"`
}

// RejectRequest es el cuerpo de POST /case-access-requests/{id}/rejection.
type RejectRequest struct {
	Motivo string `json:"motivo" example:"La atención la lleva otra área"`
}

// Created es la solicitud recién creada.
type Created struct {
	RequestID uuid.UUID                         `json:"requestId"`
	Estado    models.EstadoSolicitudVinculacion `json:"estado" example:"pendiente"`
}

// Solicitante es quien pidió el acceso.
type Solicitante struct {
	ID            uuid.UUID `json:"id"`
	NombreLaboral string    `json:"nombreLaboral"`
}

// Item es una solicitud en la bandeja. Solo metadatos: nunca datos de
// identidad de la persona ni notas.
type Item struct {
	ID              uuid.UUID                         `json:"id"`
	ExpedienteID    uuid.UUID                         `json:"expedienteId"`
	ExpedienteAlias string                            `json:"expedienteAlias"`
	Solicitante     Solicitante                       `json:"solicitante"`
	Departamento    string                            `json:"departamento"`
	DepartamentoID  uuid.UUID                         `json:"departamentoId"`
	Motivo          string                            `json:"motivo"`
	Fecha           time.Time                         `json:"fecha"`
	Estado          models.EstadoSolicitudVinculacion `json:"estado"`
}

// Page es la página de solicitudes.
type Page struct {
	Items      []Item  `json:"items"`
	NextCursor *string `json:"nextCursor"`
}
