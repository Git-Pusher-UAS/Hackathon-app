package gormrepo

import (
	"context"
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type portalCuentaRepo struct{ db *gorm.DB }

func (r *portalCuentaRepo) Create(ctx context.Context, c *models.PortalCuenta) error {
	return translate(r.db.WithContext(ctx).Create(c).Error)
}
func (r *portalCuentaRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.PortalCuenta, error) {
	return first[models.PortalCuenta](ctx, r.db, "id = ?", id)
}
func (r *portalCuentaRepo) GetByCURPNormalizada(ctx context.Context, curp string) (*models.PortalCuenta, error) {
	return first[models.PortalCuenta](ctx, r.db, "curp_normalizada = ?", curp)
}
func (r *portalCuentaRepo) FindByNombre(ctx context.Context, nombre string) (*models.PortalCuenta, error) {
	return first[models.PortalCuenta](ctx, r.db, "lower(nombre) = lower(?)", nombre)
}
func (r *portalCuentaRepo) Update(ctx context.Context, c *models.PortalCuenta) error {
	return translate(r.db.WithContext(ctx).Save(c).Error)
}

type portalContactoRepo struct{ db *gorm.DB }

func (r *portalContactoRepo) ListByCuentaID(ctx context.Context, cuentaID uuid.UUID) ([]models.PortalContactoConfianza, error) {
	return findAll[models.PortalContactoConfianza](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.Where("cuenta_id = ?", cuentaID).Order("creado_en ASC")
	})
}

type portalDesafioRepo struct{ db *gorm.DB }

func (r *portalDesafioRepo) Create(ctx context.Context, d *models.PortalDesafioOTP) error {
	return translate(r.db.WithContext(ctx).Create(d).Error)
}
func (r *portalDesafioRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.PortalDesafioOTP, error) {
	return first[models.PortalDesafioOTP](ctx, r.db, "id = ?", id)
}
func (r *portalDesafioRepo) Save(ctx context.Context, d *models.PortalDesafioOTP) error {
	return translate(r.db.WithContext(ctx).Save(d).Error)
}

type portalSesionRepo struct{ db *gorm.DB }

func (r *portalSesionRepo) Create(ctx context.Context, s *models.PortalSesion) error {
	return translate(r.db.WithContext(ctx).Create(s).Error)
}
func (r *portalSesionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.PortalSesion, error) {
	return first[models.PortalSesion](ctx, r.db, "id = ?", id)
}
func (r *portalSesionRepo) Touch(ctx context.Context, id uuid.UUID, at time.Time) error {
	return translate(r.db.WithContext(ctx).Model(&models.PortalSesion{}).Where("id = ?", id).Update("ultima_actividad", at).Error)
}
func (r *portalSesionRepo) Revoke(ctx context.Context, id uuid.UUID, at time.Time) error {
	return translate(r.db.WithContext(ctx).Model(&models.PortalSesion{}).Where("id = ? AND revocado_en IS NULL", id).Update("revocado_en", at).Error)
}

type portalAlertaRepo struct{ db *gorm.DB }

func (r *portalAlertaRepo) Create(ctx context.Context, a *models.PortalAlertaAyuda) error {
	return translate(r.db.WithContext(ctx).Create(a).Error)
}
func (r *portalAlertaRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.PortalAlertaAyuda, error) {
	return first[models.PortalAlertaAyuda](ctx, r.db, "id = ?", id)
}
func (r *portalAlertaRepo) GetByOperacionID(ctx context.Context, cuentaID, opID uuid.UUID) (*models.PortalAlertaAyuda, error) {
	return first[models.PortalAlertaAyuda](ctx, r.db, "cuenta_id = ? AND operacion_id = ?", cuentaID, opID)
}
func (r *portalAlertaRepo) Update(ctx context.Context, a *models.PortalAlertaAyuda) error {
	return translate(r.db.WithContext(ctx).Save(a).Error)
}
