package auth

import "errors"

var (
	// ErrCredenciales es el error genérico de login (no revela si el usuario existe).
	ErrCredenciales = errors.New("usuario o contraseña incorrectos")
	// ErrBloqueado indica bloqueo temporal por intentos fallidos.
	ErrBloqueado = errors.New("cuenta bloqueada temporalmente")
	// ErrDemasiadosIntentos indica rate limit por IP/usuario.
	ErrDemasiadosIntentos = errors.New("demasiados intentos de inicio de sesión")
	// ErrSuspendido indica una cuenta con estado suspendido.
	ErrSuspendido = errors.New("cuenta suspendida")
	// ErrMFARequerido indica que la cuenta tiene MFA activo (aún no implementado).
	ErrMFARequerido = errors.New("mfa_requerido")
	// ErrTokenInvalido indica un JWT ausente, alterado, vencido o con emisor incorrecto.
	ErrTokenInvalido = errors.New("token inválido o expirado")
	// ErrSesionInvalida indica una sesión revocada, vencida o inexistente.
	ErrSesionInvalida = errors.New("sesión inválida")
)
