package security

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

// SealMany cifra varios campos de un mismo registro con una \u00fanica DEK.
// Un plaintext nil produce un payload nil (campo opcional vac\u00edo).
func (k *Keyring) SealMany(
	ctx context.Context,
	claves repository.ClaveRegistroRepository,
	modulo, tipoRegistro string,
	registroID uuid.UUID,
	plaintexts, aads [][]byte,
) (uuid.UUID, [][]byte, error) {
	if len(plaintexts) != len(aads) {
		return uuid.Nil, nil, fmt.Errorf("%w: plaintexts y aads difieren", ErrCifradoInvalido)
	}
	dek := make([]byte, keySize)
	if _, err := rand.Read(dek); err != nil {
		return uuid.Nil, nil, fmt.Errorf("security: generar DEK: %w", err)
	}
	block, err := aes.NewCipher(dek)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("security: AES: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("security: GCM: %w", err)
	}

	out := make([][]byte, len(plaintexts))
	for i, pt := range plaintexts {
		if pt == nil {
			continue
		}
		nonce := make([]byte, nonceSize)
		if _, err := rand.Read(nonce); err != nil {
			return uuid.Nil, nil, fmt.Errorf("security: generar nonce: %w", err)
		}
		out[i] = append(nonce, gcm.Seal(nil, nonce, pt, aads[i])...)
	}

	wrapNonce := make([]byte, nonceSize)
	if _, err := rand.Read(wrapNonce); err != nil {
		return uuid.Nil, nil, fmt.Errorf("security: nonce de envoltura: %w", err)
	}
	kekBlock, err := aes.NewCipher(k.kek)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("security: AES KEK: %w", err)
	}
	kekGCM, err := cipher.NewGCM(kekBlock)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("security: GCM KEK: %w", err)
	}
	clave := &models.ClaveRegistro{
		ID:             uuid.New(),
		Modulo:         modulo,
		TipoRegistro:   tipoRegistro,
		RegistroID:     registroID,
		DEKEnvuelta:    kekGCM.Seal(nil, wrapNonce, dek, wrapAAD(modulo, tipoRegistro, registroID)),
		VersionKEK:     k.version,
		NonceEnvoltura: wrapNonce,
		Algoritmo:      AlgoritmoAESGCM,
	}
	if err := claves.Create(ctx, clave); err != nil {
		return uuid.Nil, nil, err
	}
	return clave.ID, out, nil
}
