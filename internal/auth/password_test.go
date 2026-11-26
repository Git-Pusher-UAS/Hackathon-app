package auth

import (
	"errors"
	"testing"
)

func TestHashPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("contraseña-segura")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" {
		t.Fatal("hash vacío")
	}

	ok, err := VerifyPassword("contraseña-segura", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatal("la contraseña correcta no verificó")
	}

	ok, err = VerifyPassword("otra-contraseña", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if ok {
		t.Fatal("una contraseña incorrecta verificó")
	}
}

func TestVerifyPasswordHashInvalido(t *testing.T) {
	if _, err := VerifyPassword("x", "no-es-un-hash-phc"); !errors.Is(err, ErrHashInvalido) {
		t.Fatalf("se esperaba ErrHashInvalido, got %v", err)
	}
}
