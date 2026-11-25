package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"time"
)

type Config struct {
	Server       ServerConfig
	DB           DBConfig
	JWT          JWTConfig
	InitialAdmin InitialAdminConfig
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

type JWTConfig struct {
	Secret    string
	ExpiresIn time.Duration
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

	cfg := &Config{
		Server: ServerConfig{
			Port:       getEnv("PORT", "3000"),
			CORSOrigin: getEnv("CORS_ORIGIN", "http://localhost:3000"),
		},
		// ("Clave del dato que busca, clave pro defecto que pondra si no lo encuentra")
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "sd_voting"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret:    getEnv("JWT_SECRET", ""),
			ExpiresIn: getEnvAsDuration("JWT_EXPIRES_IN", 24*time.Hour),
		},
		InitialAdmin: InitialAdminConfig{
			Username: getEnv("INITIAL_ADMIN_USERNAME", ""),
			Email:    getEnv("INITIAL_ADMIN_EMAIL", ""),
			Password: getEnv("INITIAL_ADMIN_PASSWORD", ""),
		},
	}

	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
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
