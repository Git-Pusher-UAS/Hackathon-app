package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// NewOpaqueToken genera un token aleatorio de 32 bytes en base64url y su hash
// SHA-256, que es lo único que se guarda en la base de datos.
func NewOpaqueToken() (token string, hash []byte, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	return token, HashOpaqueToken(token), nil
}

// HashOpaqueToken calcula el hash con el que se busca un token recibido.
func HashOpaqueToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
