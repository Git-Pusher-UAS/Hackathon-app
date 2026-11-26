package main

import (
	"context"
	"errors"
	"log"
	"strings"

	"backend-hackathon/internal/auth"
	"backend-hackathon/internal/config"
	"backend-hackathon/internal/models"
	gormrepo "backend-hackathon/internal/repository/gorm"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Comando de bootstrap:
//
//	go run ./cmd/seed
//
// Crea (de forma idempotente) el departamento, el rol admin, el permiso "admin"
// y la cuenta inicial definida en INITIAL_ADMIN_*. Requiere haber ejecutado
// antes las migraciones (go run ./cmd/migrate).
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if cfg.InitialAdmin.Username == "" || cfg.InitialAdmin.Password == "" {
		log.Fatal("INITIAL_ADMIN_USERNAME e INITIAL_ADMIN_PASSWORD son obligatorios")
	}

	db, err := gormrepo.Open(&cfg.DB)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	ctx := context.Background()

	dep, err := ensureDepartamento(db, "SISTEMAS", "Sistemas", false)
	if err != nil {
		log.Fatalf("departamento: %v", err)
	}
	rol, err := ensureRol(db, "admin", "Administrador")
	if err != nil {
		log.Fatalf("rol: %v", err)
	}
	permiso, err := ensurePermiso(db, "admin", "Acceso administrativo")
	if err != nil {
		log.Fatalf("permiso: %v", err)
	}
	if err := ensureRolPermiso(db, rol.ID, permiso.ID); err != nil {
		log.Fatalf("rol_permiso: %v", err)
	}

	// Roles del contrato SAI MVP. Se crean de forma idempotente; el rol "admin"
	// se conserva para no romper las cuentas existentes.
	rolesSpec := []struct {
		Codigo string
		Nombre string
	}{
		{"TrabajoSocial", "Trabajo Social"},
		{"Psicología", "Psicología"},
		{"Jurídico", "Jurídico"},
		{"Coordinación", "Coordinación"},
		{"Dirección", "Dirección"},
		{"AdministradorTI", "Administrador de TI"},
		{"AdministraciónAnalítica", "Administración Analítica"},
	}
	for _, item := range rolesSpec {
		if _, err := ensureRol(db, item.Codigo, item.Nombre); err != nil {
			log.Fatalf("rol %s: %v", item.Codigo, err)
		}
	}

	// El perfil de TI administra cuentas y sesiones.
	adminTI, err := ensureRol(db, "AdministradorTI", "Administrador de TI")
	if err != nil {
		log.Fatalf("rol AdministradorTI: %v", err)
	}
	if err := ensureRolPermiso(db, adminTI.ID, permiso.ID); err != nil {
		log.Fatalf("rol_permiso AdministradorTI: %v", err)
	}

	usuario := strings.ToLower(strings.TrimSpace(cfg.InitialAdmin.Username))

	var existente models.Personal
	err = db.Where("lower(usuario) = ?", usuario).First(&existente).Error
	switch {
	case err == nil:
		log.Printf("la cuenta %q ya existe; no se modifica", usuario)
		return
	case !errors.Is(err, gorm.ErrRecordNotFound):
		log.Fatalf("buscar cuenta: %v", err)
	}

	hash, err := auth.HashPassword(cfg.InitialAdmin.Password)
	if err != nil {
		log.Fatalf("hash de contraseña: %v", err)
	}

	actor := &models.Actor{Tipo: models.ActorPersonal, Activo: true}
	uow := gormrepo.NewUnitOfWork(db)

	err = uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
		if err := r.Actores().Create(ctx, actor); err != nil {
			return err
		}

		personal := &models.Personal{
			ID:                     actor.ID,
			Usuario:                usuario,
			NombreLaboral:          cfg.InitialAdmin.Username,
			RolID:                  rol.ID,
			DepartamentoID:         dep.ID,
			PasswordHash:           hash,
			RequiereCambioPassword: false,
			Estado:                 models.PersonalActivo,
			VersionPermisos:        1,
			Version:                1,
		}
		return r.Personal().Create(ctx, personal)
	})
	if err != nil {
		log.Fatalf("crear cuenta: %v", err)
	}

	// GORM aplica el DEFAULT true del modelo cuando el bool llega en cero;
	// el admin inicial no se obliga a cambiar contraseña en este bootstrap.
	if err := db.Model(&models.Personal{}).
		Where("id = ?", actor.ID).
		Update("requiere_cambio_password", false).Error; err != nil {
		log.Fatalf("actualizar cuenta: %v", err)
	}

	log.Printf("cuenta inicial creada: usuario=%q rol=%q", usuario, rol.Codigo)
}

func ensureDepartamento(db *gorm.DB, codigo, nombre string, asistencial bool) (*models.Departamento, error) {
	var d models.Departamento
	err := db.Where("codigo = ?", codigo).First(&d).Error
	if err == nil {
		return &d, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	d = models.Departamento{Codigo: codigo, Nombre: nombre, Asistencial: asistencial, Activo: true}
	if err := db.Create(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func ensureRol(db *gorm.DB, codigo, nombre string) (*models.Rol, error) {
	var r models.Rol
	err := db.Where("codigo = ?", codigo).First(&r).Error
	if err == nil {
		return &r, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	r = models.Rol{Codigo: codigo, Nombre: nombre, Activo: true}
	if err := db.Create(&r).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

func ensurePermiso(db *gorm.DB, codigo, descripcion string) (*models.Permiso, error) {
	var p models.Permiso
	err := db.Where("codigo = ?", codigo).First(&p).Error
	if err == nil {
		return &p, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	p = models.Permiso{Codigo: codigo, Descripcion: descripcion}
	if err := db.Create(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func ensureRolPermiso(db *gorm.DB, rolID, permisoID uuid.UUID) error {
	var rp models.RolPermiso
	err := db.Where("rol_id = ? AND permiso_id = ?", rolID, permisoID).First(&rp).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return db.Create(&models.RolPermiso{RolID: rolID, PermisoID: permisoID}).Error
}
