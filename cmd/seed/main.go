package main

import (
	"context"
	"errors"
	"log"
	"strings"

	"backend-hackathon/internal/auth"
	"backend-hackathon/internal/config"
	analyticsmw "backend-hackathon/internal/middleware/analytics"
	auditlogmw "backend-hackathon/internal/middleware/auditlog"
	carmw "backend-hackathon/internal/middleware/caseaccessrequest"
	casemw "backend-hackathon/internal/middleware/casefile"
	coursemw "backend-hackathon/internal/middleware/course"
	outreachmw "backend-hackathon/internal/middleware/outreach"
	refmw "backend-hackathon/internal/middleware/referral"
	surveymw "backend-hackathon/internal/middleware/survey"
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
		{"Capacitación", "Capacitación"},
	}
	roles := map[string]*models.Rol{}
	for _, item := range rolesSpec {
		r, err := ensureRol(db, item.Codigo, item.Nombre)
		if err != nil {
			log.Fatalf("rol %s: %v", item.Codigo, err)
		}
		roles[item.Codigo] = r
	}

	// El perfil de TI administra cuentas y sesiones.
	adminTI := roles["AdministradorTI"]
	if err := ensureRolPermiso(db, adminTI.ID, permiso.ID); err != nil {
		log.Fatalf("rol_permiso AdministradorTI: %v", err)
	}

	// Permisos: admin recibe todos los permisos registrados; RequirePermiso
	// también otorga bypass al permiso "admin". La matriz por rol se mantiene
	// explícita en seed para claridad operativa.
	type permisoSeed struct {
		Codigo      string
		Descripcion string
		Roles       []string
	}
	all := make([]permisoSeed, 0, 32)
	for _, p := range casemw.Permisos {
		all = append(all, permisoSeed{Codigo: p.Codigo, Descripcion: p.Descripcion})
	}
	all = append(all,
		permisoSeed{Codigo: refmw.PermRead, Descripcion: "Consultar canalizaciones", Roles: []string{"TrabajoSocial", "Psicología", "Jurídico", "Coordinación", "Dirección"}},
		permisoSeed{Codigo: refmw.PermTransition, Descripcion: "Transicionar canalizaciones", Roles: []string{"TrabajoSocial", "Psicología", "Jurídico", "Coordinación"}},
		permisoSeed{Codigo: carmw.PermRequest, Descripcion: "Solicitar acceso a expedientes", Roles: []string{"TrabajoSocial", "Psicología", "Jurídico"}},
		permisoSeed{Codigo: carmw.PermReview, Descripcion: "Revisar solicitudes de acceso", Roles: []string{"Coordinación", "Dirección"}},
		permisoSeed{Codigo: auditlogmw.PermRead, Descripcion: "Consultar bitácora de auditoría", Roles: []string{"Dirección", "AdministradorTI", "AdministraciónAnalítica"}},
		permisoSeed{Codigo: analyticsmw.PermRead, Descripcion: "Consultar analítica agregada", Roles: []string{"Coordinación", "Dirección", "AdministraciónAnalítica"}},
		permisoSeed{Codigo: analyticsmw.PermHeatmap, Descripcion: "Consultar mapa de calor", Roles: []string{"Coordinación", "Dirección", "AdministraciónAnalítica"}},
		permisoSeed{Codigo: coursemw.PermManage, Descripcion: "Gestionar cursos y bolsa de trabajo", Roles: []string{"Capacitación"}},
		permisoSeed{Codigo: outreachmw.PermManage, Descripcion: "Gestionar visitas de vinculación", Roles: []string{"Capacitación"}},
		permisoSeed{Codigo: surveymw.PermResults, Descripcion: "Consultar resultados de encuestas", Roles: []string{"Coordinación", "Dirección", "AdministraciónAnalítica", "Capacitación"}},
	)

	// Matriz RBAC base de expedientes. casefile.disclose se asigna también a
	// roles de atención (Trabajo social, Psicología, Jurídico) para "Mostrar dato".
	// Coordinación y Dirección conservan lectura/gestión según su función; no se
	// otorga acceso amplio a Administración fuera de analítica/auditoría asignada.
	porRol := map[string][]string{
		"TrabajoSocial":           {casemw.PermRead, casemw.PermWrite, casemw.PermConsent, casemw.PermRefer, casemw.PermDisclose},
		"Psicología":              {casemw.PermRead, casemw.PermWrite, casemw.PermConsent, casemw.PermRefer, casemw.PermDisclose},
		"Jurídico":                {casemw.PermRead, casemw.PermWrite, casemw.PermRefer, casemw.PermDisclose},
		"Coordinación":            {casemw.PermRead, casemw.PermWrite, casemw.PermAssign, casemw.PermDisclose, casemw.PermConsent, casemw.PermRefer},
		"Dirección":               {casemw.PermRead, casemw.PermAssign, casemw.PermDisclose},
		"AdministraciónAnalítica": {},
		"AdministradorTI":         {},
		"Capacitación":            {},
	}
	for _, p := range all {
		perm, err := ensurePermiso(db, p.Codigo, p.Descripcion)
		if err != nil {
			log.Fatalf("permiso %s: %v", p.Codigo, err)
		}
		if err := ensureRolPermiso(db, rol.ID, perm.ID); err != nil { // rol admin
			log.Fatalf("rol_permiso admin/%s: %v", p.Codigo, err)
		}
		if err := ensureRolPermiso(db, adminTI.ID, perm.ID); err != nil {
			log.Fatalf("rol_permiso AdministradorTI/%s: %v", p.Codigo, err)
		}
		for _, codRol := range p.Roles {
			porRol[codRol] = append(porRol[codRol], p.Codigo)
		}
	}
	for codRol, perms := range porRol {
		r, ok := roles[codRol]
		if !ok {
			var err error
			r, err = ensureRol(db, codRol, codRol)
			if err != nil {
				log.Fatalf("rol %s: %v", codRol, err)
			}
		}
		seen := map[string]bool{}
		for _, pc := range perms {
			if seen[pc] {
				continue
			}
			seen[pc] = true
			perm, err := ensurePermiso(db, pc, pc)
			if err != nil {
				log.Fatalf("permiso %s: %v", pc, err)
			}
			if err := ensureRolPermiso(db, r.ID, perm.ID); err != nil {
				log.Fatalf("rol_permiso %s/%s: %v", codRol, pc, err)
			}
		}
	}
	log.Print("Matriz RBAC aplicada: admin bypass en RequirePermiso + permisos explícitos por seed; casefile.disclose asignado a atención; Capacitacion con courses.manage/outreach.manage/surveys.results")

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
