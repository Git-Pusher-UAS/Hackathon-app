package gormrepo

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	defaultPageSize = 50
	maxPageSize     = 100
)

func pageLimit(p repository.PageRequest) int {
	limit := p.Limit
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}
	return limit
}

func encodeCursor(t time.Time, id uuid.UUID) string {
	raw := t.UTC().Format(time.RFC3339Nano) + "|" + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(s string) (time.Time, uuid.UUID, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, uuid.Nil, fmt.Errorf("cursor inválido")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	return t, id, nil
}

// applyPage aplica cursor, orden y límite+1 (para saber si hay página siguiente).
// Se evita la comparación de filas de Postgres y se usa una condición explícita
// para no depender de la inferencia de tipos con parámetros.
func applyPage(q *gorm.DB, p repository.PageRequest, timeCol, idCol string, desc bool) *gorm.DB {
	limit := pageLimit(p)

	if p.Cursor != "" {
		if t, id, err := decodeCursor(p.Cursor); err == nil {
			if desc {
				q = q.Where("("+timeCol+" < ?) OR ("+timeCol+" = ? AND "+idCol+" < ?)", t, t, id)
			} else {
				q = q.Where("("+timeCol+" > ?) OR ("+timeCol+" = ? AND "+idCol+" > ?)", t, t, id)
			}
		}
	}

	q = q.Limit(limit + 1)
	if desc {
		return q.Order(timeCol + " DESC, " + idCol + " DESC")
	}
	return q.Order(timeCol + " ASC, " + idCol + " ASC")
}

// cutPage recorta a `limit` y calcula el cursor de la siguiente página.
func cutPage[T any](rows []T, limit int, ts func(T) time.Time, id func(T) uuid.UUID) ([]T, string) {
	if len(rows) <= limit {
		return rows, ""
	}
	last := rows[limit-1]
	return rows[:limit], encodeCursor(ts(last), id(last))
}

// applyTimeRange aplica un intervalo semiabierto [Desde, Hasta).
func applyTimeRange(q *gorm.DB, r repository.TimeRange, col string) *gorm.DB {
	if !r.Desde.IsZero() {
		q = q.Where(col+" >= ?", r.Desde)
	}
	if !r.Hasta.IsZero() {
		q = q.Where(col+" < ?", r.Hasta)
	}
	return q
}

// first obtiene la primera fila que cumpla la condición.
func first[T any](ctx context.Context, db *gorm.DB, query string, args ...any) (*T, error) {
	var out T
	if err := db.WithContext(ctx).Where(query, args...).First(&out).Error; err != nil {
		return nil, translate(err)
	}
	return &out, nil
}

// findAll ejecuta un Find aplicando los filtros/orden indicados.
func findAll[T any](ctx context.Context, db *gorm.DB, apply func(*gorm.DB) *gorm.DB) ([]T, error) {
	q := db.WithContext(ctx)
	if apply != nil {
		q = apply(q)
	}
	var out []T
	if err := q.Find(&out).Error; err != nil {
		return nil, translate(err)
	}
	return out, nil
}
