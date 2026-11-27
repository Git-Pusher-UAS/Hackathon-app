package httpx

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"backend-hackathon/internal/security"

	"github.com/google/uuid"
)

var ErrCursorInvalido = errors.New("httpx: cursor inválido")

type cursorPayload struct {
	Actor   string `json:"a"`
	Scope   string `json:"s"`
	Filters string `json:"f"`
	Plain   string `json:"p"`
}

// FilterHash resume los filtros de un listado para ligarlos al cursor.
func FilterHash(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// EncodeCursor envuelve y firma el cursor interno del repositorio.
func EncodeCursor(secret string, actor uuid.UUID, scope, filterHash, plain string) string {
	payload, _ := json.Marshal(cursorPayload{
		Actor:   actor.String(),
		Scope:   scope,
		Filters: filterHash,
		Plain:   plain,
	})
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := security.HMAC(secret, "cursor", encoded)
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac)
}

// DecodeCursor valida firma, actor, ámbito y filtros y devuelve el cursor
// interno que entiende el repositorio.
func DecodeCursor(secret, cursor string, actor uuid.UUID, scope, filterHash string) (string, error) {
	if cursor == "" {
		return "", nil
	}
	parts := strings.SplitN(cursor, ".", 2)
	if len(parts) != 2 {
		return "", ErrCursorInvalido
	}
	expected := security.HMAC(secret, "cursor", parts[0])
	provided, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(expected, provided) {
		return "", ErrCursorInvalido
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", ErrCursorInvalido
	}
	var payload cursorPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", ErrCursorInvalido
	}
	if payload.Actor != actor.String() || payload.Scope != scope || payload.Filters != filterHash {
		return "", ErrCursorInvalido
	}
	return payload.Plain, nil
}

// Cursor liga los cursores de un listado a su ámbito y su secreto.
type Cursor struct {
	Secret string
	Scope  string
}

// Decode valida el cursor recibido contra el actor y los filtros y devuelve
// el cursor interno y el hash de filtros. Un cursor ajeno responde 422.
func (k Cursor) Decode(raw string, actor uuid.UUID, filters ...string) (string, string, error) {
	hash := FilterHash(filters...)
	plain, err := DecodeCursor(k.Secret, raw, actor, k.Scope, hash)
	if err != nil {
		return "", "", E(422, "validation_failed", "cursor inválido")
	}
	return plain, hash, nil
}

// Encode firma el siguiente cursor; nil si no hay más páginas.
func (k Cursor) Encode(actor uuid.UUID, hash, next string) *string {
	if next == "" {
		return nil
	}
	s := EncodeCursor(k.Secret, actor, k.Scope, hash, next)
	return &s
}
