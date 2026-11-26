package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Parámetros de Argon2id. Se guardan dentro del propio hash (formato PHC),
// así que se pueden endurecer más adelante sin romper hashes existentes.
const (
	argonTime    uint32 = 3
	argonMemory  uint32 = 64 * 1024 // 64 MiB
	argonThreads uint8  = 4
	argonKeyLen  uint32 = 32
	argonSaltLen        = 16
)

// ErrHashInvalido indica que el hash almacenado no tiene el formato PHC esperado.
var ErrHashInvalido = errors.New("auth: hash de contraseña inválido")

// HashPassword genera un hash Argon2id en formato PHC:
//
//	$argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: generar salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword compara password contra un hash PHC de Argon2id.
// Devuelve (false, ErrHashInvalido) si el hash almacenado está corrupto.
func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return false, ErrHashInvalido
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, ErrHashInvalido
	}
	if version != argon2.Version {
		return false, fmt.Errorf("%w: versión %d no soportada", ErrHashInvalido, version)
	}

	var (
		memory  uint32
		time    uint32
		threads uint8
	)
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, ErrHashInvalido
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ErrHashInvalido
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(want) == 0 {
		return false, ErrHashInvalido
	}

	got := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(want)))
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return false, nil
	}
	return true, nil
}

// dummySalt se usa para gastar el mismo tiempo cuando el usuario no existe
// y no filtrar por temporización si la cuenta está registrada.
var dummySalt = []byte("backend-hackathon-dummy-salt-01")

// DummyVerify ejecuta Argon2id con los mismos parámetros y descarta el resultado.
func DummyVerify(password string) {
	_ = argon2.IDKey([]byte(password), dummySalt, argonTime, argonMemory, argonThreads, argonKeyLen)
}
