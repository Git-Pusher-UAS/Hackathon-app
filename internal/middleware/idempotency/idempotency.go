// Package idempotency implementa Idempotency-Key (§1.7 del contrato) sobre
// seguridad.operaciones: repetir una creación con la misma clave en 24 h
// devuelve la misma respuesta sin volver a ejecutarla, y la misma clave con
// otro cuerpo responde 409 idempotency_mismatch.
//
// Se monta por ruta, después de AuthMiddleware. Acepta X-Operation-Id como
// alias de Idempotency-Key y deja la clave normalizada en ambos encabezados,
// para que el handler la lea con handlerutil.OperationID.
package idempotency

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"
	"backend-hackathon/internal/security"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	HeaderKey   = "Idempotency-Key"
	HeaderAlias = "X-Operation-Id"
	ttl         = 24 * time.Hour
)

// Encabezados de la respuesta original que se repiten en el reintento.
var replayHeaders = []string{fiber.HeaderETag, fiber.HeaderLocation, fiber.HeaderContentType}

type stored struct {
	Status  int               `json:"s"`
	Headers map[string]string `json:"h"`
	Body    []byte            `json:"b"`
}

// Config arma el middleware.
type Config struct {
	UOW     repository.UnitOfWork
	Keyring *security.Keyring // sin keyring solo se valida la clave
	Secret  string
	Now     func() time.Time
}

// Key lee la clave de la petición. Vacía si no viene.
func Key(c *fiber.Ctx) (string, error) {
	key := strings.TrimSpace(c.Get(HeaderKey))
	alias := strings.TrimSpace(c.Get(HeaderAlias))
	if key == "" {
		key = alias
	} else if alias != "" && !strings.EqualFold(alias, key) {
		return "", httpx.Invalid("Idempotency-Key y X-Operation-Id no coinciden", map[string]string{HeaderKey: "no_coincide"})
	}
	if key == "" {
		return "", nil
	}
	id, err := uuid.Parse(key)
	if err != nil {
		return "", httpx.Invalid("Idempotency-Key debe ser un uuid", map[string]string{HeaderKey: "formato_invalido"})
	}
	return id.String(), nil
}

// New devuelve el middleware. Con required, una petición sin clave responde 422.
func New(cfg Config, required bool) fiber.Handler {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return func(c *fiber.Ctx) error {
		key, err := Key(c)
		if err != nil {
			return err
		}
		if key == "" {
			if required {
				return httpx.Invalid("Idempotency-Key requerido", map[string]string{HeaderKey: "requerido"})
			}
			return c.Next()
		}
		c.Request().Header.Set(HeaderKey, key)
		c.Request().Header.Set(HeaderAlias, key)

		actor, ok := middleware.ActorFrom(c)
		if !ok || cfg.Keyring == nil {
			return c.Next()
		}
		opID := uuid.MustParse(key)
		tipo := "http:" + c.Method() + " " + c.Route().Path
		mac := security.HMAC(cfg.Secret, "idempotency", c.Method(), c.Path(), string(c.Body()))
		now := cfg.Now()

		prev, err := cfg.UOW.Repositories().Operaciones().Find(c.Context(), actor.ActorID, tipo, opID)
		switch {
		case err == nil:
			if !hmac.Equal(prev.PeticionHMAC, mac) {
				return httpx.E(fiber.StatusConflict, "idempotency_mismatch", "la clave ya se usó con otra petición")
			}
			if len(prev.ResultadoCifrado) > 0 && prev.ResultadoVenceEn.After(now) {
				return replay(c, cfg, prev)
			}
			// Vencida: la operación ya se aplicó; el servicio deduplica por su cuenta.
			return c.Next()
		case !errors.Is(err, repository.ErrNotFound):
			return err
		}

		if err := c.Next(); err != nil {
			return err
		}
		status := c.Response().StatusCode()
		if status < 200 || status >= 300 {
			return nil
		}
		rec := stored{Status: status, Headers: map[string]string{}, Body: append([]byte(nil), c.Response().Body()...)}
		for _, h := range replayHeaders {
			if v := string(c.Response().Header.Peek(h)); v != "" {
				rec.Headers[h] = v
			}
		}
		payload, _ := json.Marshal(rec)
		save(c, cfg, actor, tipo, opID, mac, status, payload, now)
		return nil
	}
}

func aad(id uuid.UUID) []byte {
	return []byte("seguridad.operaciones:resultado_cifrado:" + id.String())
}

// save guarda la respuesta cifrada. Si falla, la petición ya se aplicó: solo
// se pierde la repetición exacta, así que se registra y no se cambia la respuesta.
func save(c *fiber.Ctx, cfg Config, actor repository.ActorContext, tipo string, opID uuid.UUID, mac []byte, status int, payload []byte, now time.Time) {
	err := cfg.UOW.Do(c.Context(), actor, func(ctx context.Context, r repository.Repositories) error {
		id := uuid.New()
		clave, cifrado, err := cfg.Keyring.Seal(ctx, r.ClavesRegistro(), "seguridad", "operacion", id, payload, aad(id))
		if err != nil {
			return err
		}
		return r.Operaciones().Create(ctx, &models.Operacion{ID: id, ActorID: actor.ActorID, TipoOperacion: tipo,
			OperationID: opID, PeticionHMAC: mac, Estado: models.OperacionAplicada, HTTPStatus: int16(status),
			ResultadoCifrado: cifrado, ResultadoVenceEn: now.Add(ttl), AplicadaEn: now, ClaveID: clave})
	})
	if err != nil && !errors.Is(err, repository.ErrConflict) {
		log.Printf("idempotency: guardar %s: %v", tipo, err)
	}
}

func replay(c *fiber.Ctx, cfg Config, op *models.Operacion) error {
	plain, err := cfg.Keyring.Open(c.Context(), cfg.UOW.Repositories().ClavesRegistro(), op.ClaveID, op.ResultadoCifrado, aad(op.ID))
	if err != nil {
		return err
	}
	var rec stored
	if err := json.Unmarshal(plain, &rec); err != nil {
		return err
	}
	for h, v := range rec.Headers {
		c.Set(h, v)
	}
	c.Set("Idempotent-Replayed", "true")
	return c.Status(rec.Status).Send(rec.Body)
}
