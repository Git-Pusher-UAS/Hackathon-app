package httpx

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// OperationIDFromRequest acepta X-Operation-Id/Idempotency-Key en header o
// operationId/operation_id en body JSON (fallback de compatibilidad).
func OperationIDFromRequest(c *fiber.Ctx, required bool) (*uuid.UUID, error) {
	raw := strings.TrimSpace(c.Get("X-Operation-Id"))
	if raw == "" {
		raw = strings.TrimSpace(c.Get("Idempotency-Key"))
	}
	if raw == "" {
		var body map[string]json.RawMessage
		if err := json.Unmarshal(c.Body(), &body); err == nil {
			for _, k := range []string{"operationId", "operation_id"} {
				if v, ok := body[k]; ok {
					var s string
					if err := json.Unmarshal(v, &s); err == nil {
						raw = strings.TrimSpace(s)
					}
				}
			}
		}
	}
	if raw == "" {
		if required {
			return nil, E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "X-Operation-Id requerido (UUID)")
		}
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "X-Operation-Id inválido")
	}
	return &id, nil
}

// IfMatchOrVersion acepta If-Match header o version/versionTag en body JSON.
func IfMatchOrVersion(c *fiber.Ctx, secret string) (int64, error) {
	raw := c.Get("If-Match")
	if strings.TrimSpace(raw) != "" {
		tag, ok := ParseIfMatch(raw)
		if !ok {
			return 0, E(fiber.StatusPreconditionFailed, "PRECONDITION_FAILED", "If-Match inválido")
		}
		v, err := DecodeVersionTag(secret, tag)
		if err != nil {
			return 0, E(fiber.StatusPreconditionFailed, "PRECONDITION_FAILED", "versión obsoleta")
		}
		return v, nil
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(c.Body(), &body); err == nil {
		for _, k := range []string{"versionTag", "version_tag", "version"} {
			vraw, ok := body[k]
			if !ok {
				continue
			}
			var asString string
			if err := json.Unmarshal(vraw, &asString); err == nil {
				asString = strings.TrimSpace(asString)
				if asString == "" {
					continue
				}
				if n, err := uuid.Parse(asString); err == nil {
					_ = n // not a version
				}
				// try version tag
				if ver, err := DecodeVersionTag(secret, asString); err == nil {
					return ver, nil
				}
				// try numeric string
				var n int64
				if err := json.Unmarshal([]byte(asString), &n); err == nil && n > 0 {
					return n, nil
				}
			}
			var n int64
			if err := json.Unmarshal(vraw, &n); err == nil && n > 0 {
				return n, nil
			}
		}
	}
	return 0, E(fiber.StatusPreconditionRequired, "PRECONDITION_REQUIRED", "If-Match requerido")
}
