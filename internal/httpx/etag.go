package httpx

import (
	"crypto/hmac"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"

	"backend-hackathon/internal/security"
)

var ErrVersionTagInvalido = errors.New("httpx: versionTag inválido")

// EncodeVersionTag produce un tag opaco y firmado a partir de la versión.
func EncodeVersionTag(secret string, version int64) string {
	payload := "v" + strconv.FormatInt(version, 10)
	mac := security.HMAC(secret, "etag", payload)
	raw := payload + "." + base64.RawURLEncoding.EncodeToString(mac)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeVersionTag valida la firma y devuelve la versión.
func DecodeVersionTag(secret, tag string) (int64, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(tag)
	if err != nil {
		return 0, ErrVersionTagInvalido
	}
	parts := strings.SplitN(string(decoded), ".", 2)
	if len(parts) != 2 {
		return 0, ErrVersionTagInvalido
	}
	expected := security.HMAC(secret, "etag", parts[0])
	provided, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(expected, provided) {
		return 0, ErrVersionTagInvalido
	}
	if !strings.HasPrefix(parts[0], "v") {
		return 0, ErrVersionTagInvalido
	}
	version, err := strconv.ParseInt(strings.TrimPrefix(parts[0], "v"), 10, 64)
	if err != nil || version < 1 {
		return 0, ErrVersionTagInvalido
	}
	return version, nil
}

// FormatETag devuelve el valor de cabecera ETag para un versionTag.
func FormatETag(tag string) string {
	return `"` + tag + `"`
}

// ParseIfMatch extrae un versionTag de If-Match. Acepta comillas y el prefijo
// débil W/. No admite "*" ni listas: el contrato usa un único tag opaco.
func ParseIfMatch(header string) (string, bool) {
	value := strings.TrimSpace(header)
	if value == "" || value == "*" {
		return "", false
	}
	value = strings.TrimPrefix(value, "W/")
	value = strings.Trim(value, `"`)
	if value == "" || strings.Contains(value, ",") {
		return "", false
	}
	return value, true
}
