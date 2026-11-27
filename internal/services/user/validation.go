package user

import (
	"strings"
	"unicode/utf8"

	"backend-hackathon/internal/models"
	usermodels "backend-hackathon/internal/models/user"

	"github.com/google/uuid"
)

const (
	minMotivo   = 10
	maxMotivo   = 500
)

func validateCreate(in usermodels.CreateRequest) error {
	usuario := strings.TrimSpace(in.Usuario)
	if count := utf8.RuneCountInString(usuario); count < 1 || count > 100 {
		return &ValidationError{Msg: "usuario debe tener entre 1 y 100 caracteres"}
	}
	if strings.ContainsAny(usuario, " \t\r\n") {
		return &ValidationError{Msg: "usuario no puede contener espacios"}
	}

	nombre := strings.TrimSpace(in.NombreLaboral)
	if count := utf8.RuneCountInString(nombre); count < 1 || count > 150 {
		return &ValidationError{Msg: "nombreLaboral debe tener entre 1 y 150 caracteres"}
	}

	if strings.TrimSpace(in.Rol) == "" {
		return &ValidationError{Msg: "rol es obligatorio"}
	}
	if _, err := uuid.Parse(strings.TrimSpace(in.DepartamentoID)); err != nil {
		return &ValidationError{Msg: "departamentoId inválido"}
	}
	return nil
}

func validateUpdate(in usermodels.UpdateRequest) error {
	if in.Rol == nil && in.DepartamentoID == nil && in.Estado == nil {
		return &ValidationError{Msg: "no hay campos para actualizar"}
	}
	if in.Rol != nil && strings.TrimSpace(*in.Rol) == "" {
		return &ValidationError{Msg: "rol no puede estar vacío"}
	}
	if in.DepartamentoID != nil {
		if _, err := uuid.Parse(strings.TrimSpace(*in.DepartamentoID)); err != nil {
			return &ValidationError{Msg: "departamentoId inválido"}
		}
	}
	if in.Estado != nil && *in.Estado != models.PersonalActivo && *in.Estado != models.PersonalSuspendido {
		return &ValidationError{Msg: "estado debe ser activo o suspendido"}
	}
	return nil
}

func validateMotivo(motivo string) error {
	count := utf8.RuneCountInString(strings.TrimSpace(motivo))
	if count < minMotivo || count > maxMotivo {
		return &ValidationError{Msg: "motivo debe tener entre 10 y 500 caracteres"}
	}
	return nil
}
