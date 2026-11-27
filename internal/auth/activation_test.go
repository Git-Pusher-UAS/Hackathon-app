package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func (r *fakePersonalRepo) SetPassword(_ context.Context, id uuid.UUID, hash string, requiereCambio bool) error {
	p, err := r.GetByID(context.Background(), id)
	if err != nil {
		return err
	}
	p.PasswordHash = hash
	p.RequiereCambioPassword = requiereCambio
	return nil
}

func TestActivacion(t *testing.T) {
	ctx := context.Background()
	svc, repos, p := nuevoServicioDePrueba(t)
	codigo, vence, err := NuevaActivacion(ctx, repos, svc.secret, p.ID, time.Now())
	if err != nil {
		t.Fatalf("NuevaActivacion: %v", err)
	}
	if len(codigo) != 9 || codigo[4] != '-' || !vence.After(time.Now().Add(71*time.Hour)) {
		t.Fatalf("código %q vence %v", codigo, vence)
	}
	in := func(usuario, cod, pass string) ActivacionInput {
		return ActivacionInput{Usuario: usuario, Codigo: cod, Password: pass, IP: "127.0.0.1", RequestID: uuid.New()}
	}

	if err := svc.Activar(ctx, in("admin", codigo, "corta")); !errors.Is(err, ErrPasswordInvalida) {
		t.Fatalf("contraseña corta: %v", err)
	}
	if err := svc.Activar(ctx, in("otra", codigo, "NuevaClave12345")); !errors.Is(err, ErrActivacionInvalida) {
		t.Fatalf("otro usuario: %v", err)
	}
	if err := svc.Activar(ctx, in("admin", "AAAA-BBBB", "NuevaClave12345")); !errors.Is(err, ErrActivacionInvalida) {
		t.Fatalf("código inexistente: %v", err)
	}
	// El código se acepta en minúsculas y sin guion.
	if err := svc.Activar(ctx, in(" ADMIN ", strings.ToLower(codigo[:4]+codigo[5:]), "NuevaClave12345")); err != nil {
		t.Fatalf("Activar: %v", err)
	}
	if ok, _ := VerifyPassword("NuevaClave12345", p.PasswordHash); !ok || p.RequiereCambioPassword {
		t.Fatal("la contraseña nueva no quedó fijada")
	}
	if err := svc.Activar(ctx, in("admin", codigo, "OtraClave123456")); !errors.Is(err, ErrActivacionInvalida) {
		t.Fatalf("el código es de un solo uso: %v", err)
	}
}
