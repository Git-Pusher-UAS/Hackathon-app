package gormrepo

import (
	"errors"

	repository "backend-hackathon/internal/repository/repositories"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// translate convierte los errores de GORM/pgx a los centinelas del dominio.
// Las interfaces prometen ErrNotFound, ErrConflict, ErrForeignKey, etc.
func translate(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return repository.ErrNotFound
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return repository.ErrConflict
	case errors.Is(err, gorm.ErrForeignKeyViolated):
		return repository.ErrForeignKey
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return repository.ErrConflict
		case "23503": // foreign_key_violation
			return repository.ErrForeignKey
		case "23502", "23514": // not_null_violation, check_violation
			return repository.ErrConflict
		}
	}

	return err
}
