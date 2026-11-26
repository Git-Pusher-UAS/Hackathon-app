package security

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
)

type fakeClaves struct {
	byID map[uuid.UUID]*models.ClaveRegistro
}

func newFakeClaves() *fakeClaves {
	return &fakeClaves{byID: map[uuid.UUID]*models.ClaveRegistro{}}
}

func (f *fakeClaves) Create(_ context.Context, c *models.ClaveRegistro) error {
	copied := *c
	f.byID[c.ID] = &copied
	return nil
}

func (f *fakeClaves) GetByID(_ context.Context, id uuid.UUID) (*models.ClaveRegistro, error) {
	clave, ok := f.byID[id]
	if !ok {
		return nil, errors.New("no encontrada")
	}
	copied := *clave
	return &copied, nil
}

func (f *fakeClaves) GetByRegistro(_ context.Context, _ string, _ string, _ uuid.UUID) (*models.ClaveRegistro, error) {
	return nil, errors.New("no implementado")
}

func (f *fakeClaves) ListParaRotar(_ context.Context, _ int, _ int) ([]models.ClaveRegistro, error) {
	return nil, nil
}

func (f *fakeClaves) ActualizarEnvoltura(_ context.Context, _ uuid.UUID, _ []byte, _ []byte, _ int, _ time.Time) error {
	return nil
}

func newTestKeyring(t *testing.T) *Keyring {
	t.Helper()
	master := make([]byte, 32)
	for i := range master {
		master[i] = byte(i + 1)
	}
	keyring, err := NewKeyring(master, 1)
	if err != nil {
		t.Fatalf("NewKeyring: %v", err)
	}
	return keyring
}

func TestSealOpenRoundtrip(t *testing.T) {
	keyring := newTestKeyring(t)
	claves := newFakeClaves()
	ctx := context.Background()
	registroID := uuid.New()
	aad := []byte("identidad.recuperaciones_mfa:motivo:" + registroID.String())
	plaintext := []byte(`{"motivo":"recuperación presencial"}`)

	claveID, ciphertext, err := keyring.Seal(ctx, claves, "identidad", "recuperacion_mfa", registroID, plaintext, aad)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if len(ciphertext) == 0 {
		t.Fatal("Seal devolvió ciphertext vacío")
	}

	got, err := keyring.Open(ctx, claves, claveID, ciphertext, aad)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if string(got) != string(plaintext) {
		t.Fatalf("roundtrip = %q, esperado %q", got, plaintext)
	}
}

func TestOpenWithWrongAAD(t *testing.T) {
	keyring := newTestKeyring(t)
	claves := newFakeClaves()
	ctx := context.Background()
	registroID := uuid.New()
	aad := []byte("aad-correcto")

	claveID, ciphertext, err := keyring.Seal(ctx, claves, "seguridad", "operacion", registroID, []byte("secreto"), aad)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	if _, err := keyring.Open(ctx, claves, claveID, ciphertext, []byte("aad-incorrecto")); !errors.Is(err, ErrCifradoInvalido) {
		t.Fatalf("Open con AAD incorrecto = %v, esperado ErrCifradoInvalido", err)
	}
}

func TestOpenWithWrongKEK(t *testing.T) {
	keyring := newTestKeyring(t)
	claves := newFakeClaves()
	ctx := context.Background()
	registroID := uuid.New()

	claveID, ciphertext, err := keyring.Seal(ctx, claves, "seguridad", "operacion", registroID, []byte("secreto"), nil)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	otherMaster := make([]byte, 32)
	other, err := NewKeyring(otherMaster, 1)
	if err != nil {
		t.Fatalf("NewKeyring: %v", err)
	}
	if _, err := other.Open(ctx, claves, claveID, ciphertext, nil); !errors.Is(err, ErrCifradoInvalido) {
		t.Fatalf("Open con otra KEK = %v, esperado ErrCifradoInvalido", err)
	}
}

func TestNewKeyringRejectsBadSize(t *testing.T) {
	if _, err := NewKeyring(make([]byte, 16), 1); !errors.Is(err, ErrKEKInvalida) {
		t.Fatalf("NewKeyring(16 bytes) = %v, esperado ErrKEKInvalida", err)
	}
}
