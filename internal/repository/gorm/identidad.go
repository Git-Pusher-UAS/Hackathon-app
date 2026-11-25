package gormrepo

import (
	"context"
	"time"

	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---------------- A11 · identidad.actores ----------------

type actorRepo struct{ db *gorm.DB }

func (r *actorRepo) Create(ctx context.Context, actor *models.Actor) error {
	return translate(r.db.WithContext(ctx).Create(actor).Error)
}

func (r *actorRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Actor, error) {
	return first[models.Actor](ctx, r.db, "id = ?", id)
}

func (r *actorRepo) GetSistema(ctx context.Context) (*models.Actor, error) {
	return first[models.Actor](ctx, r.db, "tipo = ? AND activo = ?", models.ActorSistema, true)
}

func (r *actorRepo) SetActivo(ctx context.Context, id uuid.UUID, activo bool) error {
	res := r.db.WithContext(ctx).Model(&models.Actor{}).
		Where("id = ?", id).Update("activo", activo)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// ---------------- A12 · identidad.personal ----------------

type personalRepo struct{ db *gorm.DB }

func (r *personalRepo) Create(ctx context.Context, p *models.Personal) error {
	return translate(r.db.WithContext(ctx).Create(p).Error)
}

func (r *personalRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Personal, error) {
	return first[models.Personal](ctx, r.db, "id = ?", id)
}

func (r *personalRepo) GetByUsuario(ctx context.Context, usuario string) (*models.Personal, error) {
	return first[models.Personal](ctx, r.db, "lower(usuario) = lower(?)", usuario)
}

func (r *personalRepo) List(ctx context.Context, f repository.PersonalFilter, page repository.PageRequest) (repository.Page[models.Personal], error) {
	q := r.db.WithContext(ctx).Model(&models.Personal{})
	if f.Estado != nil {
		q = q.Where("estado = ?", *f.Estado)
	}
	if f.RolID != nil {
		q = q.Where("rol_id = ?", *f.RolID)
	}
	if f.DepartamentoID != nil {
		q = q.Where("departamento_id = ?", *f.DepartamentoID)
	}
	if f.Busqueda != "" {
		like := "%" + f.Busqueda + "%"
		q = q.Where("usuario ILIKE ? OR nombre_laboral ILIKE ?", like, like)
	}

	limit := pageLimit(page)
	q = applyPage(q, page, "creado_en", "id", false)

	var rows []models.Personal
	if err := q.Find(&rows).Error; err != nil {
		return repository.Page[models.Personal]{}, translate(err)
	}
	rows, next := cutPage(rows, limit,
		func(m models.Personal) time.Time { return m.CreadoEn },
		func(m models.Personal) uuid.UUID { return m.ID },
	)
	return repository.Page[models.Personal]{Items: rows, NextCursor: next}, nil
}

func (r *personalRepo) Update(ctx context.Context, p *models.Personal, expectedVersion int64) error {
	res := r.db.WithContext(ctx).Model(&models.Personal{}).
		Where("id = ? AND version = ?", p.ID, expectedVersion).
		Updates(map[string]any{
			"usuario":                  p.Usuario,
			"nombre_laboral":           p.NombreLaboral,
			"rol_id":                   p.RolID,
			"departamento_id":          p.DepartamentoID,
			"password_hash":            p.PasswordHash,
			"requiere_cambio_password": p.RequiereCambioPassword,
			"estado":                   p.Estado,
			"version_permisos":         p.VersionPermisos,
			"bloqueado_hasta":          p.BloqueadoHasta,
			"fallos_consecutivos":      p.FallosConsecutivos,
			"version":                  gorm.Expr("version + 1"),
			"actualizado_en":           time.Now(),
		})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrVersionMismatch
	}
	return nil
}

func (r *personalRepo) IncrementarVersionPermisos(ctx context.Context, id uuid.UUID) (int64, error) {
	var version int64
	res := r.db.WithContext(ctx).Raw(`
		UPDATE identidad.personal
		SET version_permisos = version_permisos + 1,
		    actualizado_en = now()
		WHERE id = ?
		RETURNING version_permisos`, id).Scan(&version)
	if res.Error != nil {
		return 0, translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return 0, repository.ErrNotFound
	}
	return version, nil
}

func (r *personalRepo) RegistrarFallo(ctx context.Context, id uuid.UUID, bloquearHasta *time.Time) (int, *time.Time, error) {
	var row struct {
		Fallos         int        `gorm:"column:fallos_consecutivos"`
		BloqueadoHasta *time.Time `gorm:"column:bloqueado_hasta"`
	}
	res := r.db.WithContext(ctx).Raw(`
		UPDATE identidad.personal
		SET fallos_consecutivos = fallos_consecutivos + 1,
		    bloqueado_hasta = COALESCE(?, bloqueado_hasta)
		WHERE id = ?
		RETURNING fallos_consecutivos, bloqueado_hasta`, bloquearHasta, id).Scan(&row)
	if res.Error != nil {
		return 0, nil, translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return 0, nil, repository.ErrNotFound
	}
	return row.Fallos, row.BloqueadoHasta, nil
}

func (r *personalRepo) ReiniciarFallos(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Model(&models.Personal{}).
		Where("id = ?", id).
		Updates(map[string]any{"fallos_consecutivos": 0, "bloqueado_hasta": nil})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *personalRepo) SetPassword(ctx context.Context, id uuid.UUID, hash string, requiereCambio bool) error {
	res := r.db.WithContext(ctx).Model(&models.Personal{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"password_hash":            hash,
			"requiere_cambio_password": requiereCambio,
			"actualizado_en":           time.Now(),
		})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *personalRepo) LockAdministradoresActivos(ctx context.Context) (int, error) {
	// Se bloquean las cuentas activas: es la operación que serializa los cambios
	// que podrían dejar al sistema sin ningún administrador operativo.
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).Raw(`
		SELECT id FROM identidad.personal
		WHERE estado = ?
		FOR UPDATE`, models.PersonalActivo).Scan(&ids).Error
	if err != nil {
		return 0, translate(err)
	}
	return len(ids), nil
}

// ---------------- A13 · identidad.delegaciones ----------------

type delegacionRepo struct{ db *gorm.DB }

func (r *delegacionRepo) Create(ctx context.Context, d *models.Delegacion) error {
	return translate(r.db.WithContext(ctx).Create(d).Error)
}

func (r *delegacionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Delegacion, error) {
	return first[models.Delegacion](ctx, r.db, "id = ?", id)
}

func (r *delegacionRepo) ListVigentes(ctx context.Context, personalID uuid.UUID, at time.Time) ([]models.Delegacion, error) {
	return findAll[models.Delegacion](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		return q.Where(
			"personal_id = ? AND revocado_en IS NULL AND inicio <= ? AND fin > ?",
			personalID, at, at,
		).Order("inicio ASC")
	})
}

func (r *delegacionRepo) ExisteSolapamiento(ctx context.Context, personalID, permisoID uuid.UUID, inicio, fin time.Time) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.Delegacion{}).
		Where("personal_id = ? AND permiso_id = ? AND revocado_en IS NULL AND inicio < ? AND fin > ?",
			personalID, permisoID, fin, inicio).
		Count(&n).Error
	if err != nil {
		return false, translate(err)
	}
	return n > 0, nil
}

func (r *delegacionRepo) Revocar(ctx context.Context, id uuid.UUID, at time.Time) error {
	res := r.db.WithContext(ctx).Model(&models.Delegacion{}).
		Where("id = ? AND revocado_en IS NULL", id).
		Update("revocado_en", at)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// ---------------- A14 · identidad.mfa_personal ----------------

type mfaRepo struct{ db *gorm.DB }

func (r *mfaRepo) Create(ctx context.Context, m *models.MFAPersonal) error {
	return translate(r.db.WithContext(ctx).Create(m).Error)
}

func (r *mfaRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.MFAPersonal, error) {
	return first[models.MFAPersonal](ctx, r.db, "id = ?", id)
}

func (r *mfaRepo) GetByPersonalID(ctx context.Context, personalID uuid.UUID) (*models.MFAPersonal, error) {
	return first[models.MFAPersonal](ctx, r.db, "personal_id = ?", personalID)
}

func (r *mfaRepo) Activar(ctx context.Context, id uuid.UUID, confirmadoEn time.Time, paso int64) error {
	res := r.db.WithContext(ctx).Model(&models.MFAPersonal{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"estado":            models.MFAActivo,
			"confirmado_en":     confirmadoEn,
			"ultimo_paso_usado": paso,
			"actualizado_en":    time.Now(),
		})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *mfaRepo) ConsumirPaso(ctx context.Context, id uuid.UUID, paso int64) (bool, error) {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE identidad.mfa_personal
		SET ultimo_paso_usado = ?, actualizado_en = now()
		WHERE id = ?
		  AND estado = ?
		  AND (ultimo_paso_usado IS NULL OR ultimo_paso_usado < ?)`,
		paso, id, models.MFAActivo, paso)
	if res.Error != nil {
		return false, translate(res.Error)
	}
	return res.RowsAffected > 0, nil
}

func (r *mfaRepo) Invalidar(ctx context.Context, personalID uuid.UUID) error {
	res := r.db.WithContext(ctx).Model(&models.MFAPersonal{}).
		Where("personal_id = ?", personalID).
		Update("estado", models.MFAInvalidado)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *mfaRepo) DeletePendiente(ctx context.Context, personalID uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("personal_id = ? AND estado = ?", personalID, models.MFAPendiente).
		Delete(&models.MFAPersonal{})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// ---------------- A15 · identidad.sesiones_personal ----------------

type sesionRepo struct{ db *gorm.DB }

func (r *sesionRepo) Create(ctx context.Context, s *models.SesionPersonal) error {
	return translate(r.db.WithContext(ctx).Create(s).Error)
}

func (r *sesionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.SesionPersonal, error) {
	return first[models.SesionPersonal](ctx, r.db, "id = ?", id)
}

func (r *sesionRepo) GetByTokenHash(ctx context.Context, tokenHash []byte) (*models.SesionPersonal, error) {
	return first[models.SesionPersonal](ctx, r.db, "token_hash = ?", tokenHash)
}

func (r *sesionRepo) ListByTitular(ctx context.Context, titularID uuid.UUID, soloActivas bool, now time.Time) ([]models.SesionPersonal, error) {
	return findAll[models.SesionPersonal](ctx, r.db, func(q *gorm.DB) *gorm.DB {
		q = q.Where("titular_id = ?", titularID)
		if soloActivas {
			q = q.Where("revocado_en IS NULL AND vence_absoluto > ?", now)
		}
		return q.Order("creado_en DESC")
	})
}

func (r *sesionRepo) Touch(ctx context.Context, id uuid.UUID, now time.Time) error {
	res := r.db.WithContext(ctx).Model(&models.SesionPersonal{}).
		Where("id = ?", id).Update("ultima_actividad", now)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *sesionRepo) Revocar(ctx context.Context, id uuid.UUID, at time.Time) error {
	res := r.db.WithContext(ctx).Model(&models.SesionPersonal{}).
		Where("id = ? AND revocado_en IS NULL", id).Update("revocado_en", at)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *sesionRepo) RevocarTodasDeTitular(ctx context.Context, titularID uuid.UUID, at time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Model(&models.SesionPersonal{}).
		Where("titular_id = ? AND revocado_en IS NULL", titularID).
		Update("revocado_en", at)
	return res.RowsAffected, translate(res.Error)
}

func (r *sesionRepo) PurgarAnteriores(ctx context.Context, before time.Time) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("(revocado_en IS NOT NULL AND revocado_en < ?) OR vence_absoluto < ?", before, before).
		Delete(&models.SesionPersonal{})
	return res.RowsAffected, translate(res.Error)
}

// ---------------- A16 · identidad.contextos_auth ----------------

type contextoAuthRepo struct{ db *gorm.DB }

func (r *contextoAuthRepo) Create(ctx context.Context, c *models.ContextoAuth) error {
	return translate(r.db.WithContext(ctx).Create(c).Error)
}

func (r *contextoAuthRepo) GetByTokenHash(ctx context.Context, tokenHash []byte) (*models.ContextoAuth, error) {
	return first[models.ContextoAuth](ctx, r.db, "token_hash = ?", tokenHash)
}

func (r *contextoAuthRepo) IncrementarIntentos(ctx context.Context, id uuid.UUID) (int, error) {
	var intentos int
	res := r.db.WithContext(ctx).Raw(`
		UPDATE identidad.contextos_auth
		SET intentos = intentos + 1
		WHERE id = ?
		RETURNING intentos`, id).Scan(&intentos)
	if res.Error != nil {
		return 0, translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return 0, repository.ErrNotFound
	}
	return intentos, nil
}

func (r *contextoAuthRepo) VincularEnrolamiento(ctx context.Context, id, mfaID uuid.UUID) error {
	res := r.db.WithContext(ctx).Model(&models.ContextoAuth{}).
		Where("id = ?", id).Update("enrolamiento_id", mfaID)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *contextoAuthRepo) Consumir(ctx context.Context, id uuid.UUID, at time.Time) (bool, error) {
	res := r.db.WithContext(ctx).Model(&models.ContextoAuth{}).
		Where("id = ? AND consumido_en IS NULL", id).
		Update("consumido_en", at)
	if res.Error != nil {
		return false, translate(res.Error)
	}
	return res.RowsAffected > 0, nil
}

func (r *contextoAuthRepo) PurgarVencidos(ctx context.Context, before time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Where("vence_en < ?", before).Delete(&models.ContextoAuth{})
	return res.RowsAffected, translate(res.Error)
}

// ---------------- A17 · identidad.control_intentos ----------------

type controlIntentosRepo struct{ db *gorm.DB }

func (r *controlIntentosRepo) Incrementar(ctx context.Context, ambito string, claveHMAC []byte, ventanaInicio time.Time) (*models.ControlIntentos, error) {
	var out models.ControlIntentos
	res := r.db.WithContext(ctx).Raw(`
		INSERT INTO identidad.control_intentos (ambito, clave_hmac, ventana_inicio, contador)
		VALUES (?, ?, ?, 1)
		ON CONFLICT (ambito, clave_hmac, ventana_inicio)
		DO UPDATE SET contador = identidad.control_intentos.contador + 1
		RETURNING id, ambito, clave_hmac, ventana_inicio, contador, bloqueado_hasta, creado_en`,
		ambito, claveHMAC, ventanaInicio).Scan(&out)
	if res.Error != nil {
		return nil, translate(res.Error)
	}
	return &out, nil
}

func (r *controlIntentosRepo) Bloquear(ctx context.Context, ambito string, claveHMAC []byte, ventanaInicio time.Time, hasta time.Time) error {
	res := r.db.WithContext(ctx).Model(&models.ControlIntentos{}).
		Where("ambito = ? AND clave_hmac = ? AND ventana_inicio = ?", ambito, claveHMAC, ventanaInicio).
		Update("bloqueado_hasta", hasta)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *controlIntentosRepo) EstaBloqueado(ctx context.Context, ambito string, claveHMAC []byte, now time.Time) (bool, *time.Time, error) {
	var row struct {
		BloqueadoHasta *time.Time `gorm:"column:bloqueado_hasta"`
	}
	res := r.db.WithContext(ctx).Raw(`
		SELECT bloqueado_hasta
		FROM identidad.control_intentos
		WHERE ambito = ? AND clave_hmac = ?
		  AND bloqueado_hasta IS NOT NULL AND bloqueado_hasta > ?
		ORDER BY bloqueado_hasta DESC
		LIMIT 1`, ambito, claveHMAC, now).Scan(&row)
	if res.Error != nil {
		return false, nil, translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return false, nil, nil
	}
	return true, row.BloqueadoHasta, nil
}

func (r *controlIntentosRepo) PurgarAnteriores(ctx context.Context, before time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Where("ventana_inicio < ?", before).Delete(&models.ControlIntentos{})
	return res.RowsAffected, translate(res.Error)
}

// ---------------- A18 · identidad.recuperaciones_mfa ----------------

type recuperacionMFARepo struct{ db *gorm.DB }

func (r *recuperacionMFARepo) Create(ctx context.Context, rec *models.RecuperacionMFA) error {
	return translate(r.db.WithContext(ctx).Create(rec).Error)
}

func (r *recuperacionMFARepo) GetByID(ctx context.Context, id uuid.UUID) (*models.RecuperacionMFA, error) {
	return first[models.RecuperacionMFA](ctx, r.db, "id = ?", id)
}

func (r *recuperacionMFARepo) GetPendiente(ctx context.Context, personalID uuid.UUID) (*models.RecuperacionMFA, error) {
	var out models.RecuperacionMFA
	err := r.db.WithContext(ctx).
		Where("personal_id = ? AND completado_en IS NULL", personalID).
		Order("creado_en DESC").
		First(&out).Error
	if err != nil {
		return nil, translate(err)
	}
	return &out, nil
}

func (r *recuperacionMFARepo) Completar(ctx context.Context, id uuid.UUID, at time.Time) error {
	res := r.db.WithContext(ctx).Model(&models.RecuperacionMFA{}).
		Where("id = ? AND completado_en IS NULL", id).
		Update("completado_en", at)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}
