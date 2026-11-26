package user

import "errors"

var (
	// ErrNotFound indica que el recurso no existe o está fuera del ámbito.
	ErrNotFound = errors.New("usuario: recurso no encontrado")
	// ErrDuplicateUsername indica que el usuario ya está registrado.
	ErrDuplicateUsername = errors.New("usuario: nombre de usuario repetido")
	// ErrLastAdmin impide dejar el sistema sin administradores activos.
	ErrLastAdmin = errors.New("usuario: no se puede dejar el sistema sin administrador activo")
	// ErrSelfElevation impide que una cuenta modifique sus propios permisos.
	ErrSelfElevation = errors.New("usuario: autoelevación prohibida")
	// ErrVersionMismatch indica un If-Match obsoleto.
	ErrVersionMismatch = errors.New("usuario: versión obsoleta")
	// ErrPreconditionRequired indica que falta If-Match.
	ErrPreconditionRequired = errors.New("usuario: If-Match requerido")
	// ErrMFAVerificationRequired indica que no se declaró la verificación presencial.
	ErrMFAVerificationRequired = errors.New("usuario: verificación presencial requerida")
	// ErrUnauthenticated indica una sesión o cuenta inválida.
	ErrUnauthenticated = errors.New("usuario: sesión inválida")
	// ErrCryptoUnavailable indica que la KEK no está configurada.
	ErrCryptoUnavailable = errors.New("usuario: cifrado no configurado")
)

// ValidationError describe un campo inválido sin acoplarse a HTTP.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }
