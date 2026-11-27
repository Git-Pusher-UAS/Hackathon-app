// Package caseaccessrequest define los DTOs de /api/case-access-requests.
package caseaccessrequest

import (
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

type CreateRequest struct {
	LinkToken string `json:"linkToken"`
	Motivo    string `json:"motivo"`
}

type RejectRequest struct {
	Motivo string `json:"motivo"`
}

type Created struct {
	RequestID uuid.UUID                         `json:"requestId"`
	Estado    models.EstadoSolicitudVinculacion `json:"estado"`
}

type Solicitante struct {
	ID            uuid.UUID `json:"id"`
	NombreLaboral string    `json:"nombreLaboral"`
}

type Item struct {
	ID           uuid.UUID                         `json:"id"`
	ExpedienteID uuid.UUID                         `json:"expedienteId"`
	Solicitante  Solicitante                       `json:"solicitante"`
	Departamento string                            `json:"departamento"`
	Motivo       string                            `json:"motivo"`
	Fecha        time.Time                         `json:"fecha"`
	Estado       models.EstadoSolicitudVinculacion `json:"estado"`
}
