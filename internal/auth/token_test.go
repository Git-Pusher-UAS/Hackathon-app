package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTokenRoundTrip(t *testing.T) {
	personalID := uuid.New()
	sessionID := uuid.New()

	token, expires, err := SignToken("secreto-de-prueba", time.Hour, personalID, Claims{
		ActorID:     personalID,
		SessionID:   sessionID,
		Rol:         "admin",
		Permisos:    []string{"admin"},
		PermVersion: 3,
	})
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	if token == "" || expires.IsZero() {
		t.Fatal("token o expiración vacíos")
	}

	claims, err := ParseToken("secreto-de-prueba", token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if claims.Subject != personalID.String() {
		t.Fatalf("sub = %q, esperado %q", claims.Subject, personalID)
	}
	if claims.SessionID != sessionID {
		t.Fatalf("sid = %v, esperado %v", claims.SessionID, sessionID)
	}
	if claims.ActorID != personalID || claims.Rol != "admin" || claims.PermVersion != 3 {
		t.Fatalf("claims inesperados: %+v", claims)
	}
	if len(claims.Permisos) != 1 || claims.Permisos[0] != "admin" {
		t.Fatalf("permisos inesperados: %v", claims.Permisos)
	}
}

func TestTokenFirmaIncorrecta(t *testing.T) {
	token, _, err := SignToken("secreto-bueno", time.Hour, uuid.New(), Claims{
		ActorID:   uuid.New(),
		SessionID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}

	if _, err := ParseToken("secreto-malo", token); err == nil {
		t.Fatal("se esperaba error con otra clave")
	}
}
