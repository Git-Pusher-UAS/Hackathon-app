// Package caseaccessrequest define los DTOs de /case-access-requests (API-018/019).
package caseaccessrequest

import (
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

// CreateRequest pide acceso a un expediente existente con el linkToken que
// devolvió POST /case-files/duplicate-checks.
type CreateRequest struct {
	LinkToken string `json:"linkToken"`
	Motivo    string `json:"motivo" example:"Atención psicológica solicitada por la persona"`
}

// Created es la solicitud recién creada.
type Created struct {
	ID     uuid.UUID                         `json:"id"`
	Estado models.EstadoSolicitudVinculacion `json:"estado" example:"pendiente"`
}

// Ref es una referencia con nombre visible.
type Ref struct {
	ID     uuid.UUID `json:"id"`
	Nombre string    `json:"nombre"`
}

// Item es una solicitud en la bandeja. Solo metadatos: ningún dato de
// identidad de la persona ni notas.
type Item struct {
	ID           uuid.UUID                         `json:"id"`
	ExpedienteID uuid.UUID                         `json:"expedienteId"`
	Solicitante  Ref                               `json:"solicitante"`
	Departamento Ref                               `json:"departamento"`
	Motivo       string                            `json:"motivo"`
	Estado       models.EstadoSolicitudVinculacion `json:"estado"`
	CreatedAt    time.Time                         `json:"createdAt"`
	ResueltaEn   *time.Time                        `json:"resueltaEn"`
}

// Page es la página de solicitudes.
type Page struct {
	Items      []Item  `json:"items"`
	NextCursor *string `json:"nextCursor"`
}
