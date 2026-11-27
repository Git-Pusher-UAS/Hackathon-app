package config

import (
	"encoding/base64"
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env          string
	Server       ServerConfig
	DB           DBConfig
	JWT          JWTConfig
	KEK          KEKConfig
	InitialAdmin InitialAdminConfig
	MFA          MFAConfig
	CURPKey      string
}

type ServerConfig struct {
	Port       string
	CORSOrigin string
}

// datos que debe conseguir del -env
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}
type MFAConfig struct {
	Secret string
}

type JWTConfig struct {
	Secret    string
	ExpiresIn time.Duration
}

// KEKConfig contiene la clave maestra externa usada para envolver las DEK de
// los registros cifrados. Master queda vacío si KEK_MASTER no está definida;
// los comandos de migración/seed no la necesitan, la API sí.
type KEKConfig struct {
	Master  []byte
	Version int
}

type InitialAdminConfig struct {
	Username string
	Email    string
	Password string
}

func Load() (*Config, error) {
	//busca el .env
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: .env file not found, using system environment variables")
	}

	kekMaster, err := decodeKEK(getEnv("KEK_MASTER", ""))
	if err != nil {
		return nil, err
	}
	kekVersion := getEnvAsInt("KEK_VERSION", 1)
	if kekVersion < 1 {
		return nil, fmt.Errorf("KEK_VERSION debe ser mayor que cero")
	}

	cfg := &Config{
		Env: strings.ToLower(strings.TrimSpace(getEnv("APP_ENV", "production"))),
		Server: ServerConfig{
			Port:       getEnv("PORT", "3000"),
			CORSOrigin: getEnv("CORS_ORIGIN", "http://localhost:3000"),
		},
		CURPKey: getEnv("CURP_HMAC_KEY", ""),
		// ("Clave del dato que busca, clave pro defecto que pondra si no lo encuentra")
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5433"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "sd_voting"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret:    getEnv("JWT_SECRET", ""),
			ExpiresIn: getEnvAsDuration("JWT_EXPIRES_IN", 12*time.Hour),
		},
		MFA: MFAConfig{
			Secret: getEnv("MFA_ENCRYPTION_KEY", ""),
		},
		KEK: KEKConfig{
			Master:  kekMaster,
			Version: kekVersion,
		},
		InitialAdmin: InitialAdminConfig{
			Username: getEnv("INITIAL_ADMIN_USERNAME", ""),
			Email:    getEnv("INITIAL_ADMIN_EMAIL", ""),
			Password: getEnv("INITIAL_ADMIN_PASSWORD", ""),
		},
	}
	if cfg.MFA.Secret == "" {
		return nil, fmt.Errorf("MFA_ENCRYPTION_KEY es obligatoria")
	}
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

// IsDev indica si la API corre en modo desarrollo local (APP_ENV=dev|development|local).
func (c *Config) IsDev() bool {
	switch strings.ToLower(strings.TrimSpace(c.Env)) {
	case "dev", "development", "local":
		return true
	default:
		return false
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func decodeKEK(raw string) ([]byte, error) {
	if raw == "" {
		return nil, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("KEK_MASTER no es base64 válido: %w", err)
	}
	if len(decoded) != 32 {
		return nil, fmt.Errorf("KEK_MASTER debe decodificar a 32 bytes, tiene %d", len(decoded))
	}
	return decoded, nil
}
