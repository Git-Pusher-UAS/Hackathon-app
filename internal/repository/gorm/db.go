// Package gormrepo contiene las implementaciones GORM de los puertos definidos
// en internal/repository/repositories. El dominio (services/handlers) nunca debe
// importar este paquete: solo conoce las interfaces.
package gormrepo

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"backend-hackathon/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open abre un pool gestionado por GORM contra Postgres.
//
// Notas:
//   - Se usa PreferSimpleProtocol porque Neon expone un pooler (PgBouncer) y los
//     prepared statements pueden fallar o desincronizarse.
//   - TranslateError hace que GORM traduzca violaciones de Postgres a sus errores
//     centinela (ErrDuplicatedKey, ErrForeignKeyViolated), que luego mapeamos a los
//     del paquete repository.
func Open(cfg *config.DBConfig) (*gorm.DB, error) {
	connURL := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     net.JoinHostPort(cfg.Host, cfg.Port),
		Path:     cfg.Name,
		RawQuery: url.Values{"sslmode": {cfg.SSLMode}}.Encode(),
	}

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  connURL.String(),
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Warn),
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("gorm open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("gorm sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return db, nil
}
