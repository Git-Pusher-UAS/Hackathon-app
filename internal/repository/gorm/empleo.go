package gormrepo

import (
	"context"
	"time"

	"backend-hackathon/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type empleadorRepo struct{ db *gorm.DB }

func (r *empleadorRepo) Create(ctx context.Context, e *models.Empleador) error {
	return translate(r.db.WithContext(ctx).Create(e).Error)
}
func (r *empleadorRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Empleador, error) {
	return first[models.Empleador](ctx, r.db, "id = ?", id)
}
func (r *empleadorRepo) GetByCorreo(ctx context.Context, correo string) (*models.Empleador, error) {
	return first[models.Empleador](ctx, r.db, "lower(correo) = lower(?)", correo)
}
func (r *empleadorRepo) List(ctx context.Context, estado *models.EstadoRegistroEmpleador) ([]models.Empleador, error) {
	q := r.db.WithContext(ctx).Model(&models.Empleador{})
	if estado != nil {
		q = q.Where("estado = ?", *estado)
	}
	var rows []models.Empleador
	return rows, translate(q.Order("creado_en DESC").Find(&rows).Error)
}
func (r *empleadorRepo) Save(ctx context.Context, e *models.Empleador) error {
	return translate(r.db.WithContext(ctx).Save(e).Error)
}

type empleadorDesafioRepo struct{ db *gorm.DB }

func (r *empleadorDesafioRepo) Create(ctx context.Context, d *models.EmpleadorDesafioOTP) error {
	return translate(r.db.WithContext(ctx).Create(d).Error)
}
func (r *empleadorDesafioRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.EmpleadorDesafioOTP, error) {
	return first[models.EmpleadorDesafioOTP](ctx, r.db, "id = ?", id)
}
func (r *empleadorDesafioRepo) Save(ctx context.Context, d *models.EmpleadorDesafioOTP) error {
	return translate(r.db.WithContext(ctx).Save(d).Error)
}

type empleadorSesionRepo struct{ db *gorm.DB }

func (r *empleadorSesionRepo) Create(ctx context.Context, s *models.EmpleadorSesion) error {
	return translate(r.db.WithContext(ctx).Create(s).Error)
}
func (r *empleadorSesionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.EmpleadorSesion, error) {
	return first[models.EmpleadorSesion](ctx, r.db, "id = ?", id)
}
func (r *empleadorSesionRepo) Touch(ctx context.Context, id uuid.UUID, at time.Time) error {
	return translate(r.db.WithContext(ctx).Model(&models.EmpleadorSesion{}).Where("id = ?", id).Update("ultima_actividad", at).Error)
}
func (r *empleadorSesionRepo) Revoke(ctx context.Context, id uuid.UUID, at time.Time) error {
	return translate(r.db.WithContext(ctx).Model(&models.EmpleadorSesion{}).Where("id = ? AND revocado_en IS NULL", id).Update("revocado_en", at).Error)
}

type cvRepo struct{ db *gorm.DB }

func (r *cvRepo) GetPerfil(ctx context.Context, cuentaID uuid.UUID) (*models.CVPerfil, error) {
	return first[models.CVPerfil](ctx, r.db, "cuenta_id = ?", cuentaID)
}
func (r *cvRepo) UpsertPerfil(ctx context.Context, p *models.CVPerfil) error {
	var existing models.CVPerfil
	err := r.db.WithContext(ctx).Where("cuenta_id = ?", p.CuentaID).First(&existing).Error
	if err == nil {
		p.ID = existing.ID
		p.CreadoEn = existing.CreadoEn
		return translate(r.db.WithContext(ctx).Save(p).Error)
	}
	if err != gorm.ErrRecordNotFound {
		return translate(err)
	}
	return translate(r.db.WithContext(ctx).Create(p).Error)
}
func (r *cvRepo) ListExperiencias(ctx context.Context, cuentaID uuid.UUID) ([]models.CVExperiencia, error) {
	var rows []models.CVExperiencia
	return rows, translate(r.db.WithContext(ctx).Where("cuenta_id = ?", cuentaID).Order("inicio DESC").Find(&rows).Error)
}
func (r *cvRepo) ListEstudios(ctx context.Context, cuentaID uuid.UUID) ([]models.CVEstudio, error) {
	var rows []models.CVEstudio
	return rows, translate(r.db.WithContext(ctx).Where("cuenta_id = ?", cuentaID).Order("inicio DESC").Find(&rows).Error)
}
func (r *cvRepo) ReplaceExperiencias(ctx context.Context, cuentaID uuid.UUID, items []models.CVExperiencia) error {
	if err := r.db.WithContext(ctx).Where("cuenta_id = ?", cuentaID).Delete(&models.CVExperiencia{}).Error; err != nil {
		return translate(err)
	}
	if len(items) == 0 {
		return nil
	}
	return translate(r.db.WithContext(ctx).Create(&items).Error)
}
func (r *cvRepo) ReplaceEstudios(ctx context.Context, cuentaID uuid.UUID, items []models.CVEstudio) error {
	if err := r.db.WithContext(ctx).Where("cuenta_id = ?", cuentaID).Delete(&models.CVEstudio{}).Error; err != nil {
		return translate(err)
	}
	if len(items) == 0 {
		return nil
	}
	return translate(r.db.WithContext(ctx).Create(&items).Error)
}
