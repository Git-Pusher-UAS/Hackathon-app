package gormrepo

import (
	"context"

	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var _ repository.UnitOfWork = (*unitOfWork)(nil)

type unitOfWork struct {
	db       *gorm.DB
	newStore func(*gorm.DB) repository.Repositories
}

// NewUnitOfWork construye el UnitOfWork sobre la conexión raíz.
func NewUnitOfWork(db *gorm.DB) repository.UnitOfWork {
	return &unitOfWork{db: db, newStore: NewRepositories}
}

// Do ejecuta fn en una transacción con el actor establecido para RLS.
func (u *unitOfWork) Do(
	ctx context.Context,
	actor repository.ActorContext,
	fn func(context.Context, repository.Repositories) error,
) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := setConfig(tx, "app.actor_id", actor.ActorID.String()); err != nil {
			return err
		}
		if err := setConfig(tx, "app.request_id", actor.RequestID.String()); err != nil {
			return err
		}
		if actor.IPOrigen != nil {
			if err := setConfig(tx, "app.ip_origen", *actor.IPOrigen); err != nil {
				return err
			}
		}
		if err := setConfig(tx, "app.is_system", "false"); err != nil {
			return err
		}
		return fn(ctx, u.newStore(tx))
	})
}

// DoSystem ejecuta fn sin actor de personal (bootstrap, login, directorio público,
// tareas internas). Deja app.is_system = true para que las políticas RLS lo traten
// como proceso del sistema.
func (u *unitOfWork) DoSystem(
	ctx context.Context,
	fn func(context.Context, repository.Repositories) error,
) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := setConfig(tx, "app.actor_id", uuid.Nil.String()); err != nil {
			return err
		}
		if err := setConfig(tx, "app.is_system", "true"); err != nil {
			return err
		}
		return fn(ctx, u.newStore(tx))
	})
}

// DoIndependent abre una transacción separada de la que esté en curso.
// Sirve para auditar una denegación aunque la transacción de negocio haga rollback.
func (u *unitOfWork) DoIndependent(
	ctx context.Context,
	fn func(context.Context, repository.Repositories) error,
) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ctx, u.newStore(tx))
	})
}

// Repositories devuelve repositorios sin transacción explícita (lecturas simples).
func (u *unitOfWork) Repositories() repository.Repositories {
	return u.newStore(u.db)
}

// setConfig fija una variable local de sesión para RLS.
// Nota: SET no acepta parámetros; se usa set_config(..., true) (true = local a la tx).
func setConfig(tx *gorm.DB, key, value string) error {
	return translate(tx.Exec("SELECT set_config(?, ?, true)", key, value).Error)
}
