package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Issuer identifica los tokens emitidos por este servicio.
const Issuer = "backend-hackathon"

// Claims es el contenido del JWT de acceso.
type Claims struct {
	ActorID     uuid.UUID `json:"act"`
	SessionID   uuid.UUID `json:"sid"`
	Rol         string    `json:"rol"`
	Permisos    []string  `json:"perms"`
	PermVersion int64     `json:"pv"`
	jwt.RegisteredClaims
}

// SignToken firma un JWT HS256 para el personal indicado.
func SignToken(secret string, ttl time.Duration, personalID uuid.UUID, claims Claims) (string, time.Time, error) {
	now := time.Now()
	expires := now.Add(ttl)

	claims.RegisteredClaims = jwt.RegisteredClaims{
		Subject:   personalID.String(),
		Issuer:    Issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expires),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth: firmar token: %w", err)
	}
	return signed, expires, nil
}

// ParseToken valida la firma, el método, el emisor y la expiración (con 30s de
// tolerancia de reloj) y devuelve los claims.
func ParseToken(secret, raw string) (*Claims, error) {
	claims := &Claims{}

	parsed, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("auth: método de firma no permitido: %s", t.Method.Alg())
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(Issuer),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil || !parsed.Valid {
		return nil, ErrTokenInvalido
	}

	if claims.Subject == "" || claims.SessionID == uuid.Nil {
		return nil, ErrTokenInvalido
	}

	return claims, nil
}

// RandomToken devuelve n bytes aleatorios codificados en base64url.
func RandomToken(n int) (string, error) {
	return randomToken(n)
}

// randomToken devuelve n bytes aleatorios codificados en base64url.
func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: generar token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
