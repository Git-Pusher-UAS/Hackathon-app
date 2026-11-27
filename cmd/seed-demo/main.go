package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"backend-hackathon/internal/auth"
	"backend-hackathon/internal/config"
	"backend-hackathon/internal/models"
	casemodels "backend-hackathon/internal/models/casefile"
	gormrepo "backend-hackathon/internal/repository/gorm"
	repository "backend-hackathon/internal/repository/repositories"
	"backend-hackathon/internal/security"
	caseservice "backend-hackathon/internal/services/casefile"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const demoStaffPassword = "Demo1234!"

var (
	// Portal usuarias
	usuaria1ID = uuid.MustParse("11111111-1111-4111-8111-111111111101")
	usuaria2ID = uuid.MustParse("11111111-1111-4111-8111-111111111102")

	// Empleadores y vacantes
	empleador1ID = uuid.MustParse("22222222-2222-4222-8222-222222222201")
	empleador2ID = uuid.MustParse("22222222-2222-4222-8222-222222222202")
	vacante1ID   = uuid.MustParse("22222222-2222-4222-8222-222222222211")
	vacante2ID   = uuid.MustParse("22222222-2222-4222-8222-222222222212")
	vacante3ID   = uuid.MustParse("22222222-2222-4222-8222-222222222213")

	// Cursos y visitas de vinculación
	curso1ID  = uuid.MustParse("33333333-3333-4333-8333-333333333301")
	curso2ID  = uuid.MustParse("33333333-3333-4333-8333-333333333302")
	curso3ID  = uuid.MustParse("33333333-3333-4333-8333-333333333303")
	visita1ID = uuid.MustParse("33333333-3333-4333-8333-333333333311")
	visita2ID = uuid.MustParse("33333333-3333-4333-8333-333333333312")

	// Encuesta, preguntas y envíos anónimos (K=5)
	encuestaID  = uuid.MustParse("44444444-4444-4444-8444-444444444401")
	pregunta1ID = uuid.MustParse("44444444-4444-4444-8444-444444444411")
	pregunta2ID = uuid.MustParse("44444444-4444-4444-8444-444444444412")
	pregunta3ID = uuid.MustParse("44444444-4444-4444-8444-444444444413")
	envioIDs    = []uuid.UUID{
		uuid.MustParse("44444444-4444-4444-8444-444444444421"),
		uuid.MustParse("44444444-4444-4444-8444-444444444422"),
		uuid.MustParse("44444444-4444-4444-8444-444444444423"),
		uuid.MustParse("44444444-4444-4444-8444-444444444424"),
		uuid.MustParse("44444444-4444-4444-8444-444444444425"),
	}
)

// Comando de datos simulados para desarrollo local:
//
//	APP_ENV=dev go run ./cmd/seed-demo
//
// Requiere haber ejecutado previamente:
//
//	go run ./cmd/migrate
//	go run ./cmd/seed
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if !cfg.IsDev() {
		log.Fatalf("seed-demo solo puede ejecutarse en modo desarrollo (APP_ENV=dev|development|local); actual=%q", cfg.Env)
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
	uow := gormrepo.NewUnitOfWork(db)
	now := time.Now().UTC()

	// 1. Catálogos base
	deps, sectores, tipos, insts, propIDs, err := seedCatalogs(db, now)
	if err != nil {
		log.Fatalf("catalogs: %v", err)
	}

	// 2. Cuentas de personal (requiere roles de cmd/seed)
	staff, err := seedStaff(ctx, db, uow, deps)
	if err != nil {
		log.Fatalf("staff: %v", err)
	}

	// 3. Cuentas del portal, contactos de confianza y CV
	if err := seedPortal(db); err != nil {
		log.Fatalf("portal: %v", err)
	}

	// 4. Empleadores, vacantes y postulación
	if err := seedBolsa(db, now); err != nil {
		log.Fatalf("bolsa: %v", err)
	}

	// 5. Cursos, inscripción y visitas de vinculación
	capStaff := staff["cap.demo"]
	if err := seedCapacitacion(db, capStaff.ID, sectores["SEC-CENTRO"].ID, sectores["SEC-NORTE"].ID, now); err != nil {
		log.Fatalf("capacitacion: %v", err)
	}

	// 6. Encuesta activa, 2 asignaciones pendientes y 5 envíos anónimos (K=5)
	if err := seedEncuestas(db, now); err != nil {
		log.Fatalf("encuestas: %v", err)
	}

	// 7. 5 expedientes cifrados (K=5 para analítica/mapa), notas, asignación y canalizaciones
	if len(cfg.KEK.Master) == 0 || cfg.CURPKey == "" {
		log.Println("warning: KEK_MASTER o CURP_HMAC_KEY no configuradas; se omiten los expedientes cifrados")
	} else {
		keyring, err := security.NewKeyring(cfg.KEK.Master, cfg.KEK.Version)
		if err != nil {
			log.Fatalf("kek: %v", err)
		}
		caseSvc := caseservice.New(uow, keyring, cfg.JWT.Secret, cfg.CURPKey)
		if err := seedCaseFiles(ctx, caseSvc, staff, deps, sectores["SEC-CENTRO"].ID, tipos["violencia_familiar"].ID, insts, propIDs, now); err != nil {
			log.Fatalf("case-files: %v", err)
		}
	}

	log.Println("seed-demo completado correctamente (idempotente)")
}

func seedCatalogs(db *gorm.DB, now time.Time) (
	deps map[string]*models.Departamento,
	sectores map[string]*models.Sector,
	tipos map[string]*models.TipoCaso,
	insts map[string]*models.Institucion,
	propIDs []string,
	err error,
) {
	deps = map[string]*models.Departamento{}
	depSpecs := []struct {
		codigo, nombre string
		asistencial    bool
	}{
		{"TRABAJO_SOCIAL", "Trabajo Social", true},
		{"PSICOLOGIA", "Psicología", true},
		{"JURIDICO", "Jurídico", true},
		{"CAPACITACION", "Capacitación", false},
		{"DIRECCION", "Dirección", false},
	}
	for _, s := range depSpecs {
		d, err := ensureDepartamento(db, s.codigo, s.nombre, s.asistencial)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		deps[s.codigo] = d
	}

	sectores = map[string]*models.Sector{}
	geo := json.RawMessage(`{"type":"Polygon","coordinates":[[[-107.41,24.79],[-107.38,24.79],[-107.38,24.82],[-107.41,24.82],[-107.41,24.79]]]}`)
	secSpecs := []struct {
		codigo, nombre string
		tipo           models.TipoSector
	}{
		{"SEC-CENTRO", "Sector Centro", models.SectorSector},
		{"SEC-NORTE", "Colonia Norte", models.SectorColonia},
		{"SEC-SUR", "Sindicatura Sur", models.SectorSindicatura},
	}
	for _, s := range secSpecs {
		sec, err := ensureSector(db, s.codigo, s.nombre, s.tipo, geo)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		sectores[s.codigo] = sec
	}

	tipos = map[string]*models.TipoCaso{}
	tipoSpecs := []struct{ codigo, nombre string }{
		{"por_clasificar", "Por clasificar"},
		{"violencia_familiar", "Violencia familiar"},
		{"apoyo_psicologico", "Apoyo psicológico"},
		{"orientacion_juridica", "Orientación jurídica"},
	}
	for _, s := range tipoSpecs {
		tc, err := ensureTipoCaso(db, s.codigo, s.nombre)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		tipos[s.codigo] = tc
	}

	insts = map[string]*models.Institucion{}
	instSpecs := []struct {
		nombre, tel, horario, ruta, desc string
		prioritario                      bool
	}{
		{"Línea de Emergencias 911", "911", "24 horas, todos los días", "Atención inmediata de emergencias", "Emergencias policiales y médicas", true},
		{"Línea de Atención a las Mujeres", "800-822-7956", "24 horas, todos los días", "Orientación telefónica y canalización inmediata", "Atención gratuita y confidencial", true},
		{"Fiscalía Especializada en Atención a Mujeres", "667-713-3200", "Lunes a viernes 08:00 a 18:00", "Denuncia penal y medidas de protección", "Acompañamiento ministerial especializado", false},
	}
	for _, s := range instSpecs {
		inst, err := ensureInstitucion(db, s.nombre, s.tel, s.horario, s.ruta, s.desc, s.prioritario, now)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		insts[s.nombre] = inst
	}

	aviso, err := ensureAvisoPrivacidad(db, "v1.0-demo", "Aviso de privacidad integral para atención, seguimiento y vinculación del Sistema de Atención Integral (SAI).", now.Add(-24*time.Hour))
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	propSpecs := []struct {
		codigo, desc        string
		opcional, revocable bool
	}{
		{"atencion_integral", "Atención integral y registro de expediente", false, false},
		{"seguimiento_caso", "Seguimiento interdisciplinario y canalizaciones", false, true},
		{"vinculacion_bolsa", "Vinculación a capacitación y bolsa de trabajo", true, true},
	}
	for _, s := range propSpecs {
		p, err := ensureProposito(db, s.codigo, s.desc, s.opcional, s.revocable)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		if err := ensureAvisoProposito(db, aviso.ID, p.ID); err != nil {
			return nil, nil, nil, nil, nil, err
		}
		propIDs = append(propIDs, p.ID.String())
	}

	return deps, sectores, tipos, insts, propIDs, nil
}

func seedStaff(ctx context.Context, db *gorm.DB, uow repository.UnitOfWork, deps map[string]*models.Departamento) (map[string]*models.Personal, error) {
	specs := []struct {
		usuario, nombre, rolCodigo, depCodigo string
	}{
		{"ts.demo", "Laura Trabajo Social Demo", "TrabajoSocial", "TRABAJO_SOCIAL"},
		{"psi.demo", "Sofía Psicología Demo", "Psicología", "PSICOLOGIA"},
		{"jur.demo", "Valeria Jurídico Demo", "Jurídico", "JURIDICO"},
		{"coord.demo", "Claudia Coordinación Demo", "Coordinación", "TRABAJO_SOCIAL"},
		{"cap.demo", "Andrea Capacitación Demo", "Capacitación", "CAPACITACION"},
		{"dir.demo", "Beatriz Dirección Demo", "Dirección", "DIRECCION"},
	}

	hash, err := auth.HashPassword(demoStaffPassword)
	if err != nil {
		return nil, err
	}

	out := make(map[string]*models.Personal, len(specs))
	for _, s := range specs {
		var rol models.Rol
		if err := db.Where("codigo = ?", s.rolCodigo).First(&rol).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("falta el rol base %q; ejecuta primero: go run ./cmd/seed", s.rolCodigo)
			}
			return nil, err
		}

		var existing models.Personal
		err := db.Where("lower(usuario) = ?", strings.ToLower(s.usuario)).First(&existing).Error
		if err == nil {
			out[s.usuario] = &existing
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		dep := deps[s.depCodigo]
		actor := &models.Actor{Tipo: models.ActorPersonal, Activo: true}
		var created models.Personal
		err = uow.DoSystem(ctx, func(ctx context.Context, r repository.Repositories) error {
			if err := r.Actores().Create(ctx, actor); err != nil {
				return err
			}
			created = models.Personal{
				ID:                     actor.ID,
				Usuario:                s.usuario,
				NombreLaboral:          s.nombre,
				RolID:                  rol.ID,
				DepartamentoID:         dep.ID,
				PasswordHash:           hash,
				RequiereCambioPassword: false,
				Estado:                 models.PersonalActivo,
				VersionPermisos:        1,
				Version:                1,
			}
			return r.Personal().Create(ctx, &created)
		})
		if err != nil {
			return nil, err
		}
		if err := db.Model(&models.Personal{}).Where("id = ?", actor.ID).Update("requiere_cambio_password", false).Error; err != nil {
			return nil, err
		}
		created.RequiereCambioPassword = false
		out[s.usuario] = &created
	}
	return out, nil
}

func seedPortal(db *gorm.DB) error {
	accounts := []struct {
		id           uuid.UUID
		alias        string
		nombre       string
		curp         string
		telefono     string
		contactName  string
		contactPhone string
	}{
		{usuaria1ID, "Usuaria-Demo01", "Ana Morales Demo", "MOLA920514MSLRNS01", "6671112233", "Rosa Morales", "6679990011"},
		{usuaria2ID, "Usuaria-Demo02", "María Torres Demo", "TOMA881123MSLRTN02", "6672223344", "Elena Torres", "6678880022"},
	}

	for _, a := range accounts {
		var existing models.PortalCuenta
		err := db.Where("curp_normalizada = ?", a.curp).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row := models.PortalCuenta{
				ID:          a.id,
				Alias:       a.alias,
				Nombre:      a.nombre,
				CURPNormalizada: a.curp,
				Telefono:    a.telefono,
				AceptaAviso: true,
			}
			if err := db.Create(&row).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			if existing.ID != a.id {
				return fmt.Errorf("portal cuenta %s ya existe con id distinto (%s)", a.curp, existing.ID)
			}
		}

		var cnt int64
		if err := db.Model(&models.PortalContactoConfianza{}).Where("cuenta_id = ?", a.id).Count(&cnt).Error; err != nil {
			return err
		}
		if cnt == 0 {
			c := models.PortalContactoConfianza{CuentaID: a.id, Nombre: a.contactName, Telefono: a.contactPhone}
			if err := db.Create(&c).Error; err != nil {
				return err
			}
		}
	}

	// CV completo solo para Usuaria 1 (Usuaria 2 queda sin CV para validar error de postulación)
	var perfil models.CVPerfil
	if err := db.Where("cuenta_id = ?", usuaria1ID).First(&perfil).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		p := models.CVPerfil{
			CuentaID:    usuaria1ID,
			Nombre:      "Ana Morales Demo",
			Titular:     "Auxiliar Administrativa",
			Resumen:     "Experiencia en atención al cliente y control administrativo.",
			Contacto:    "6671112233",
			Habilidades: `["Atención al cliente","Excel básico","Administración"]`,
		}
		if err := db.Create(&p).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	var expCnt int64
	if err := db.Model(&models.CVExperiencia{}).Where("cuenta_id = ?", usuaria1ID).Count(&expCnt).Error; err != nil {
		return err
	}
	if expCnt == 0 {
		if err := db.Create(&models.CVExperiencia{
			CuentaID:    usuaria1ID,
			Empresa:     "Papelería del Centro",
			Puesto:      "Auxiliar de oficina",
			Inicio:      "2022-03",
			Fin:         "2025-08",
			Descripcion: "Facturación, atención telefónica y control de inventario.",
		}).Error; err != nil {
			return err
		}
	}

	var estCnt int64
	if err := db.Model(&models.CVEstudio{}).Where("cuenta_id = ?", usuaria1ID).Count(&estCnt).Error; err != nil {
		return err
	}
	if estCnt == 0 {
		if err := db.Create(&models.CVEstudio{
			CuentaID:    usuaria1ID,
			Institucion: "CBTis 224",
			Grado:       "Técnico en Administración",
			Inicio:      "2018-08",
			Fin:         "2021-06",
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedBolsa(db *gorm.DB, now time.Time) error {
	employers := []models.Empleador{
		{
			ID:          empleador1ID,
			Nombre:      "Comercializadora Pacífico Demo",
			Correo:      "rrhh@empresa-demo.mx",
			Telefono:    "6673334455",
			Descripcion: "Empresa regional de comercio y servicios administrativos.",
			Estado:      models.EmpleadorAprobado,
		},
		{
			ID:          empleador2ID,
			Nombre:      "Taller Textil Creativo Demo",
			Correo:      "contacto@taller-demo.mx",
			Telefono:    "6674445566",
			Descripcion: "Taller local de confección en revisión administrativa.",
			Estado:      models.EmpleadorPendiente,
		},
	}
	for _, e := range employers {
		var existing models.Empleador
		err := db.Where("lower(correo) = lower(?)", e.Correo).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&e).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	empID := empleador1ID
	jobs := []models.Vacante{
		{
			ID:            vacante1ID,
			EmpleadorID:   &empID,
			Puesto:        "Auxiliar Administrativo",
			Empresa:       "Comercializadora Pacífico Demo",
			Sector:        "Administración",
			Horario:       "Matutino",
			TipoEmpleo:    "Tiempo completo",
			Ubicacion:     "Centro, Culiacán",
			Descripcion:   "Apoyo en archivo, agenda, captura de datos y atención telefónica.",
			Requisitos:    "Preparatoria terminada, manejo básico de hojas de cálculo.",
			Estado:        models.VacantePublicada,
			PublicadaEn:   now.Add(-48 * time.Hour),
			ActualizadaEn: now.Add(-48 * time.Hour),
		},
		{
			ID:            vacante2ID,
			EmpleadorID:   &empID,
			Puesto:        "Atención a Clientes",
			Empresa:       "Comercializadora Pacífico Demo",
			Sector:        "Comercio",
			Horario:       "Vespertino",
			TipoEmpleo:    "Medio tiempo",
			Ubicacion:     "Colonia Norte, Culiacán",
			Descripcion:   "Orientación a clientes en sucursal y seguimiento de pedidos.",
			Requisitos:    "Facilidad de palabra y disponibilidad vespertina.",
			Estado:        models.VacantePublicada,
			PublicadaEn:   now.Add(-24 * time.Hour),
			ActualizadaEn: now.Add(-24 * time.Hour),
		},
		{
			ID:            vacante3ID,
			EmpleadorID:   &empID,
			Puesto:        "Recepcionista",
			Empresa:       "Comercializadora Pacífico Demo",
			Sector:        "Servicios",
			Horario:       "Matutino",
			TipoEmpleo:    "Tiempo completo",
			Ubicacion:     "Centro, Culiacán",
			Descripcion:   "Recepción de visitantes y canalización de llamadas.",
			Requisitos:    "Buena presentación y organización.",
			Estado:        models.VacantePendiente,
			PublicadaEn:   now.Add(-12 * time.Hour),
			ActualizadaEn: now.Add(-12 * time.Hour),
		},
	}
	for _, v := range jobs {
		var existing models.Vacante
		err := db.Where("id = ?", v.ID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&v).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	var post models.Postulacion
	err := db.Where("vacante_id = ? AND usuaria_id = ?", vacante1ID, usuaria1ID).First(&post).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		p := models.Postulacion{
			VacanteID: vacante1ID,
			UsuariaID: usuaria1ID,
			Estado:    models.PostulacionEnviada,
		}
		if err := db.Create(&p).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func seedCapacitacion(db *gorm.DB, capActorID, sectorCentroID, sectorNorteID uuid.UUID, now time.Time) error {
	baseDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	cursos := []models.Curso{
		{
			ID:          curso1ID,
			Tipo:        models.CursoCurso,
			Titulo:      "Finanzas Personales y Emprendimiento",
			Descripcion: "Taller práctico de presupuesto familiar, ahorro y arranque de microemprendimientos.",
			Fecha:       baseDate.AddDate(0, 0, 14),
			Hora:        "10:00",
			Lugar:       "Aula 1 - Centro",
			Cupo:        20,
			CreadoPor:   capActorID,
		},
		{
			ID:          curso2ID,
			Tipo:        models.CursoCharla,
			Titulo:      "Derechos Laborales y Seguridad Social",
			Descripcion: "Charla informativa sobre contratos, prestaciones y defensa de derechos laborales.",
			Fecha:       baseDate.AddDate(0, 0, 21),
			Hora:        "12:00",
			Lugar:       "Auditorio Municipal",
			Cupo:        35,
			CreadoPor:   capActorID,
		},
		{
			ID:          curso3ID,
			Tipo:        models.CursoConferencia,
			Titulo:      "Autonomía Económica y Liderazgo",
			Descripcion: "Conferencia magistral sobre redes de apoyo, liderazgo comunitario y empleo digno.",
			Fecha:       baseDate.AddDate(0, 0, 30),
			Hora:        "16:00",
			Lugar:       "Salón Principal",
			Cupo:        50,
			CreadoPor:   capActorID,
		},
	}
	for _, c := range cursos {
		var existing models.Curso
		err := db.Where("id = ?", c.ID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&c).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	var ins models.Inscripcion
	err := db.Where("curso_id = ? AND usuaria_id = ?", curso1ID, usuaria1ID).First(&ins).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.Create(&models.Inscripcion{
			CursoID:   curso1ID,
			UsuariaID: usuaria1ID,
			Alias:     "Participante 1",
			Presente:  false,
		}).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	visitas := []models.Visita{
		{
			ID:          visita1ID,
			SectorID:    sectorCentroID,
			Fecha:       baseDate.AddDate(0, 0, 10),
			Tema:        "Difusión de servicios y prevención en comercio local",
			Responsable: "Andrea Capacitación Demo",
			Estado:      models.VisitaPlaneada,
			CreadoPor:   capActorID,
		},
		{
			ID:          visita2ID,
			SectorID:    sectorNorteID,
			Fecha:       baseDate.AddDate(0, 0, -5),
			Tema:        "Jornada informativa en colonia Norte",
			Responsable: "Andrea Capacitación Demo",
			Estado:      models.VisitaRealizada,
			CreadoPor:   capActorID,
		},
	}
	for _, v := range visitas {
		var existing models.Visita
		err := db.Where("id = ?", v.ID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&v).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}

func seedEncuestas(db *gorm.DB, now time.Time) error {
	var enc models.Encuesta
	err := db.Where("id = ?", encuestaID).First(&enc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.Create(&models.Encuesta{
			ID:      encuestaID,
			Titulo:  "Encuesta de satisfacción de atención",
			Minutos: 3,
			Activa:  true,
		}).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	optsJSON := json.RawMessage(`["Excelente","Buena","Regular"]`)
	preguntas := []models.PreguntaEncuesta{
		{
			ID:         pregunta1ID,
			EncuestaID: encuestaID,
			Clave:      "q1",
			Orden:      1,
			Texto:      "¿Qué tan clara fue la atención recibida? (1 a 5)",
			Tipo:       models.PreguntaEscala,
		},
		{
			ID:         pregunta2ID,
			EncuestaID: encuestaID,
			Clave:      "q2",
			Orden:      2,
			Texto:      "¿Cómo calificas el trato del personal?",
			Tipo:       models.PreguntaOpcion,
			Opciones:   optsJSON,
		},
		{
			ID:         pregunta3ID,
			EncuestaID: encuestaID,
			Clave:      "q3",
			Orden:      3,
			Texto:      "¿Tienes algún comentario o sugerencia?",
			Tipo:       models.PreguntaAbierta,
		},
	}
	for _, p := range preguntas {
		var existing models.PreguntaEncuesta
		err := db.Where("encuesta_id = ? AND clave = ?", p.EncuestaID, p.Clave).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&p).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	for _, uid := range []uuid.UUID{usuaria1ID, usuaria2ID} {
		var asig models.AsignacionEncuesta
		err := db.Where("encuesta_id = ? AND usuaria_id = ?", encuestaID, uid).First(&asig).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&models.AsignacionEncuesta{
				EncuestaID: encuestaID,
				UsuariaID:  uid,
				Contestada: false,
			}).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	scores := []float64{5, 4, 5, 4, 5}
	comments := []string{
		"Excelente acompañamiento desde el primer contacto.",
		"Atención muy amable y clara.",
		"Me orientaron rápido sobre los pasos a seguir.",
		"Buen seguimiento y trato respetuoso.",
		"Muy útil la información de cursos y bolsa.",
	}
	for i, envioID := range envioIDs {
		var cnt int64
		if err := db.Model(&models.RespuestaEncuesta{}).Where("envio_id = ?", envioID).Count(&cnt).Error; err != nil {
			return err
		}
		if cnt > 0 {
			continue
		}
		score := scores[i]
		opt := "Excelente"
		comment := comments[i]
		rows := []models.RespuestaEncuesta{
			{EncuestaID: encuestaID, PreguntaID: pregunta1ID, EnvioID: envioID, Numero: &score, CreadoEn: now},
			{EncuestaID: encuestaID, PreguntaID: pregunta2ID, EnvioID: envioID, Texto: &opt, CreadoEn: now},
			{EncuestaID: encuestaID, PreguntaID: pregunta3ID, EnvioID: envioID, Texto: &comment, CreadoEn: now},
		}
		if err := db.Create(&rows).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedCaseFiles(
	ctx context.Context,
	caseSvc caseservice.Service,
	staff map[string]*models.Personal,
	deps map[string]*models.Departamento,
	sectorCentroID uuid.UUID,
	tipoViolenciaID uuid.UUID,
	insts map[string]*models.Institucion,
	propIDs []string,
	now time.Time,
) error {
	ts := staff["ts.demo"]
	jur := staff["jur.demo"]
	coord := staff["coord.demo"]

	tsActor := repository.ActorContext{ActorID: ts.ID, RequestID: uuid.New()}
	coordActor := repository.ActorContext{ActorID: coord.ID, RequestID: uuid.New()}

	secIDStr := sectorCentroID.String()
	tipoIDStr := tipoViolenciaID.String()
	instLineaIDStr := insts["Línea de Atención a las Mujeres"].ID.String()

	consent := casemodels.ConsentInput{
		Decision:     models.DecisionAceptado,
		Modalidad:    models.ModalidadPresencial,
		PropositoIDs: propIDs,
		CapturadoEn:  now.Add(-2 * time.Hour),
	}

	strPtr := func(s string) *string { return &s }

	caseSpecs := []struct {
		opID uuid.UUID
		req  casemodels.CreateRequest
	}{
		{
			opID: uuid.MustParse("55555555-5555-4555-8555-555555555501"),
			req: casemodels.CreateRequest{
				Alias:           "Caso-Demo-01",
				Nombre:          "Lucía Vega Demo",
				CURP:            "VELU900310MSLGVR01",
				FechaNacimiento: "1990-03-10",
				Telefono:        strPtr("6675001001"),
				Domicilio:       strPtr("Colonia Centro, Culiacán"),
				SectorID:        secIDStr,
				OrigenContacto:  models.OrigenPresencial,
				TipoCasoID:      &tipoIDStr,
				Consentimiento:  consent,
			},
		},
		{
			opID: uuid.MustParse("55555555-5555-4555-8555-555555555502"),
			req: casemodels.CreateRequest{
				Alias:           "Caso-Demo-02",
				Nombre:          "Sofía Castro Demo",
				CURP:            "CASO870722MSLSTN02",
				FechaNacimiento: "1987-07-22",
				Telefono:        strPtr("6675001002"),
				Domicilio:       strPtr("Sector Centro, Culiacán"),
				SectorID:        secIDStr,
				OrigenContacto:  models.OrigenLineaAtencion,
				TipoCasoID:      &tipoIDStr,
				Consentimiento:  consent,
			},
		},
		{
			opID: uuid.MustParse("55555555-5555-4555-8555-555555555503"),
			req: casemodels.CreateRequest{
				Alias:           "Caso-Demo-03",
				Nombre:          "Elena Ruiz Demo",
				CURP:            "RUEL950115MSLZRD03",
				FechaNacimiento: "1995-01-15",
				Telefono:        strPtr("6675001003"),
				Domicilio:       strPtr("Zona Centro, Culiacán"),
				SectorID:        secIDStr,
				OrigenContacto:  models.OrigenPresencial,
				TipoCasoID:      &tipoIDStr,
				Consentimiento:  consent,
			},
		},
		{
			opID: uuid.MustParse("55555555-5555-4555-8555-555555555504"),
			req: casemodels.CreateRequest{
				Alias:               "Caso-Demo-04",
				Nombre:              "Carmen Díaz Demo",
				CURP:                "DICA910908MSLZCN04",
				FechaNacimiento:     "1991-09-08",
				Telefono:            strPtr("6675001004"),
				Domicilio:           strPtr("Colonia Centro, Culiacán"),
				SectorID:            secIDStr,
				OrigenContacto:      models.OrigenInstitucion,
				InstitucionOrigenID: &instLineaIDStr,
				TipoCasoID:          &tipoIDStr,
				Consentimiento:      consent,
			},
		},
		{
			opID: uuid.MustParse("55555555-5555-4555-8555-555555555505"),
			req: casemodels.CreateRequest{
				Alias:           "Caso-Demo-05",
				Nombre:          "Patricia Navarro Demo",
				CURP:            "NAPA840630MSLVRT05",
				FechaNacimiento: "1984-06-30",
				Telefono:        strPtr("6675001005"),
				Domicilio:       strPtr("Sector Centro, Culiacán"),
				SectorID:        secIDStr,
				OrigenContacto:  models.OrigenPatrullaGenero,
				TipoCasoID:      &tipoIDStr,
				Consentimiento:  consent,
			},
		},
	}

	summaries := make([]*casemodels.Summary, 0, len(caseSpecs))
	for _, cs := range caseSpecs {
		s, err := caseSvc.Create(ctx, tsActor, cs.opID, cs.req)
		if err != nil {
			return fmt.Errorf("create %s: %w", cs.req.Alias, err)
		}
		summaries = append(summaries, s)
	}

	depTS := deps["TRABAJO_SOCIAL"].ID
	depPsiStr := deps["PSICOLOGIA"].ID.String()
	instFiscaliaStr := insts["Fiscalía Especializada en Atención a Mujeres"].ID.String()

	// Notas iniciales en casos 1, 2 y 3 (idempotentes por opID)
	notes := []struct {
		caseID uuid.UUID
		opID   uuid.UUID
		texto  string
	}{
		{summaries[0].ID, uuid.MustParse("55555555-5555-4555-8555-555555555511"), "Entrevista inicial de Trabajo Social realizada; se acuerda canalización psicológica."},
		{summaries[1].ID, uuid.MustParse("55555555-5555-4555-8555-555555555512"), "Atención telefónica inicial; caso pasa a seguimiento con apoyo jurídico y canalización externa."},
		{summaries[2].ID, uuid.MustParse("55555555-5555-4555-8555-555555555513"), "Registro de primera entrevista y orientación sobre red de apoyo."},
	}
	for _, n := range notes {
		if _, err := caseSvc.CreateNote(ctx, tsActor, n.caseID, depTS, n.opID, casemodels.NoteCreateRequest{
			Texto:      n.texto,
			OcurridoEn: now.Add(-1 * time.Hour),
		}); err != nil {
			return fmt.Errorf("create note: %w", err)
		}
	}

	// Caso 1: canalización interna pendiente hacia Psicología
	if _, err := caseSvc.CreateReferral(ctx, tsActor, summaries[0].ID, uuid.MustParse("55555555-5555-4555-8555-555555555521"), casemodels.ReferralCreateRequest{
		Tipo:      string(models.CanalizacionInternaTipo),
		DestinoID: &depPsiStr,
		Resumen:   "Se solicita valoración psicológica inicial por situación de violencia familiar.",
	}); err != nil {
		return fmt.Errorf("internal referral: %w", err)
	}

	// Caso 2: pasar a en_seguimiento, asignar a jur.demo y crear canalización externa
	if summaries[1].Estado == models.ExpedienteAbierto {
		nextEstado := models.ExpedienteEnSeguimiento
		motivo := "Inicio de seguimiento interdisciplinario"
		if _, err := caseSvc.Update(ctx, tsActor, summaries[1].ID, casemodels.UpdateRequest{
			Estado: &nextEstado,
			Motivo: &motivo,
		}, summaries[1].VersionTag); err != nil {
			return fmt.Errorf("update caso 2: %w", err)
		}
	}
	if _, err := caseSvc.Assign(ctx, coordActor, summaries[1].ID, casemodels.AssignRequest{
		PersonalID:     jur.ID.String(),
		DepartamentoID: deps["JURIDICO"].ID.String(),
		Motivo:         "Asignación inicial a Jurídico para seguimiento legal demo",
	}); err != nil && !errors.Is(err, caseservice.ErrAlreadyAssigned) {
		return fmt.Errorf("assign caso 2: %w", err)
	}
	if _, err := caseSvc.CreateReferral(ctx, tsActor, summaries[1].ID, uuid.MustParse("55555555-5555-4555-8555-555555555522"), casemodels.ReferralCreateRequest{
		Tipo:          string(models.CanalizacionExternaTipo),
		InstitucionID: &instFiscaliaStr,
		Resumen:       "Canalización externa para acompañamiento y denuncia ante Fiscalía Especializada.",
	}); err != nil {
		return fmt.Errorf("external referral: %w", err)
	}

	return nil
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

func ensureSector(db *gorm.DB, codigo, nombre string, tipo models.TipoSector, geo json.RawMessage) (*models.Sector, error) {
	var s models.Sector
	err := db.Where("codigo = ?", codigo).First(&s).Error
	if err == nil {
		return &s, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	s = models.Sector{
		Codigo:             codigo,
		Nombre:             nombre,
		Tipo:               tipo,
		Geometria:          geo,
		VersionCartografia: "2026-demo",
		Activo:             true,
	}
	if err := db.Create(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func ensureTipoCaso(db *gorm.DB, codigo, nombre string) (*models.TipoCaso, error) {
	var tc models.TipoCaso
	err := db.Where("codigo = ?", codigo).First(&tc).Error
	if err == nil {
		return &tc, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	tc = models.TipoCaso{Codigo: codigo, Nombre: nombre, Activo: true}
	if err := db.Create(&tc).Error; err != nil {
		return nil, err
	}
	return &tc, nil
}

func ensureInstitucion(db *gorm.DB, nombre, tel, horario, ruta, desc string, prioritario bool, now time.Time) (*models.Institucion, error) {
	var inst models.Institucion
	err := db.Where("nombre = ?", nombre).First(&inst).Error
	if err == nil {
		return &inst, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	inst = models.Institucion{
		Nombre:       nombre,
		Telefono:     tel,
		Horario:      horario,
		RutaAtencion: ruta,
		Descripcion:  desc,
		Prioritario:  prioritario,
		Publicable:   true,
		VerificadoEn: &now,
		Activo:       true,
		Version:      1,
	}
	if err := db.Create(&inst).Error; err != nil {
		return nil, err
	}
	return &inst, nil
}

func ensureAvisoPrivacidad(db *gorm.DB, codigoVersion, texto string, publicadoEn time.Time) (*models.AvisoPrivacidad, error) {
	var a models.AvisoPrivacidad
	err := db.Where("codigo_version = ?", codigoVersion).First(&a).Error
	if err == nil {
		return &a, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	a = models.AvisoPrivacidad{
		CodigoVersion: codigoVersion,
		Texto:         texto,
		PublicadoEn:   publicadoEn,
	}
	if err := db.Create(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func ensureProposito(db *gorm.DB, codigo, desc string, opcional, revocable bool) (*models.Proposito, error) {
	var p models.Proposito
	err := db.Where("codigo = ?", codigo).First(&p).Error
	if err == nil {
		return &p, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	p = models.Proposito{
		Codigo:      codigo,
		Descripcion: desc,
		Opcional:    opcional,
		Activo:      true,
		Revocable:   revocable,
	}
	if err := db.Create(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func ensureAvisoProposito(db *gorm.DB, avisoID, propositoID uuid.UUID) error {
	var ap models.AvisoProposito
	err := db.Where("aviso_id = ? AND proposito_id = ?", avisoID, propositoID).First(&ap).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return db.Create(&models.AvisoProposito{AvisoID: avisoID, PropositoID: propositoID}).Error
}
