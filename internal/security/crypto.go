// Package security implementa el cifrado de sobre (envelope encryption) que
// usa el modelo relacional: una KEK externa envuelve una DEK por registro y la
// DEK cifra el contenido. La KEK nunca se guarda en la base de datos.
package security

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"

	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

const (
	// AlgoritmoAESGCM es el único algoritmo soportado por el MVP.
	AlgoritmoAESGCM = "AES-256-GCM"

	keySize   = 32
	nonceSize = 12
)

var (
	// ErrKEKInvalida indica que la clave maestra no tiene el tamaño requerido.
	ErrKEKInvalida = errors.New("security: KEK inválida")
	// ErrCifradoInvalido indica un payload o una clave envuelta corruptos.
	ErrCifradoInvalido = errors.New("security: cifrado inválido")
)

// Keyring contiene la KEK activa y su versión.
type Keyring struct {
	kek     []byte
	version int
}

// NewKeyring valida y construye el llavero de cifrado.
func NewKeyring(master []byte, version int) (*Keyring, error) {
	if len(master) != keySize {
		return nil, fmt.Errorf("%w: se esperaban %d bytes, llegaron %d", ErrKEKInvalida, keySize, len(master))
	}
	if version < 1 {
		return nil, fmt.Errorf("%w: versión %d", ErrKEKInvalida, version)
	}
	copied := make([]byte, len(master))
	copy(copied, master)
	return &Keyring{kek: copied, version: version}, nil
}

// Version devuelve la versión de KEK activa.
func (k *Keyring) Version() int { return k.version }

// Seal cifra plaintext con una DEK nueva y guarda la DEK envuelta en
// seguridad.claves_registro. Devuelve el id de la clave y el payload cifrado.
//
// El payload tiene el formato nonce || ciphertext. El AAD debe vincular el
// registro y la columna; se usa también al abrir.
func (k *Keyring) Seal(
	ctx context.Context,
	claves repository.ClaveRegistroRepository,
	modulo string,
	tipoRegistro string,
	registroID uuid.UUID,
	plaintext []byte,
	aad []byte,
) (uuid.UUID, []byte, error) {
	dek := make([]byte, keySize)
	if _, err := rand.Read(dek); err != nil {
		return uuid.Nil, nil, fmt.Errorf("security: generar DEK: %w", err)
	}

	dataNonce := make([]byte, nonceSize)
	if _, err := rand.Read(dataNonce); err != nil {
		return uuid.Nil, nil, fmt.Errorf("security: generar nonce: %w", err)
	}

	dataBlock, err := aes.NewCipher(dek)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("security: AES: %w", err)
	}
	dataGCM, err := cipher.NewGCM(dataBlock)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("security: GCM: %w", err)
	}
	ciphertext := dataGCM.Seal(nil, dataNonce, plaintext, aad)
	payload := make([]byte, 0, len(dataNonce)+len(ciphertext))
	payload = append(payload, dataNonce...)
	payload = append(payload, ciphertext...)

	wrapNonce := make([]byte, nonceSize)
	if _, err := rand.Read(wrapNonce); err != nil {
		return uuid.Nil, nil, fmt.Errorf("security: generar nonce de envoltura: %w", err)
	}

	kekBlock, err := aes.NewCipher(k.kek)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("security: AES KEK: %w", err)
	}
	kekGCM, err := cipher.NewGCM(kekBlock)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("security: GCM KEK: %w", err)
	}
	wrappedDEK := kekGCM.Seal(nil, wrapNonce, dek, wrapAAD(modulo, tipoRegistro, registroID))

	clave := &models.ClaveRegistro{
		ID:             uuid.New(),
		Modulo:         modulo,
		TipoRegistro:   tipoRegistro,
		RegistroID:     registroID,
		DEKEnvuelta:    wrappedDEK,
		VersionKEK:     k.version,
		NonceEnvoltura: wrapNonce,
		Algoritmo:      AlgoritmoAESGCM,
	}
	if err := claves.Create(ctx, clave); err != nil {
		return uuid.Nil, nil, err
	}

	return clave.ID, payload, nil
}

// Open descifra un payload producido por Seal.
func (k *Keyring) Open(
	ctx context.Context,
	claves repository.ClaveRegistroRepository,
	claveID uuid.UUID,
	payload []byte,
	aad []byte,
) ([]byte, error) {
	clave, err := claves.GetByID(ctx, claveID)
	if err != nil {
		return nil, err
	}
	if clave.VersionKEK != k.version {
		return nil, fmt.Errorf("%w: versión KEK %d no disponible", ErrCifradoInvalido, clave.VersionKEK)
	}
	if clave.Algoritmo != AlgoritmoAESGCM {
		return nil, fmt.Errorf("%w: algoritmo %q", ErrCifradoInvalido, clave.Algoritmo)
	}

	kekBlock, err := aes.NewCipher(k.kek)
	if err != nil {
		return nil, fmt.Errorf("security: AES KEK: %w", err)
	}
	kekGCM, err := cipher.NewGCM(kekBlock)
	if err != nil {
		return nil, fmt.Errorf("security: GCM KEK: %w", err)
	}
	dek, err := kekGCM.Open(nil, clave.NonceEnvoltura, clave.DEKEnvuelta, wrapAAD(clave.Modulo, clave.TipoRegistro, clave.RegistroID))
	if err != nil {
		return nil, fmt.Errorf("%w: desenvolver DEK: %w", ErrCifradoInvalido, err)
	}

	if len(payload) < nonceSize {
		return nil, fmt.Errorf("%w: payload demasiado corto", ErrCifradoInvalido)
	}
	dataBlock, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("security: AES: %w", err)
	}
	dataGCM, err := cipher.NewGCM(dataBlock)
	if err != nil {
		return nil, fmt.Errorf("security: GCM: %w", err)
	}
	plaintext, err := dataGCM.Open(nil, payload[:nonceSize], payload[nonceSize:], aad)
	if err != nil {
		return nil, fmt.Errorf("%w: descifrar payload: %w", ErrCifradoInvalido, err)
	}
	return plaintext, nil
}

// HMAC calcula un HMAC-SHA256 canónico sobre las partes separadas por "|".
// Se usa para firmar cursores, ETags y la huella de operaciones.
func HMAC(secret string, parts ...string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	for _, part := range parts {
		_, _ = mac.Write([]byte(part))
		_, _ = mac.Write([]byte("|"))
	}
	return mac.Sum(nil)
}

func wrapAAD(modulo, tipoRegistro string, registroID uuid.UUID) []byte {
	return []byte(modulo + ":" + tipoRegistro + ":" + registroID.String())
}
