package gormrepo

import (
	"fmt"

	"backend-hackathon/internal/models"

	"gorm.io/gorm"
)

// schemas son los esquemas Postgres que usa el modelo relacional.
// AutoMigrate NO crea esquemas, así que se crean antes de migrar las tablas.
var schemas = []string{
	"catalogos",
	"identidad",
	"seguridad",
	"expedientes",
	"consentimiento",
	"canalizacion",
	"auditoria",
	"ayuda",
	"analitica",
	"capacitacion",
}

// Tables devuelve los modelos que representan TABLAS reales.
//
// Importante: NO incluye los modelos de internal/models/vistas.go
// (TimelineItem, EstadoProposito, ResumenFila, MapaCelda), porque son
// proyecciones de solo lectura. AutoMigrate intentaría crear tablas con esos
// nombres. Las vistas deben crearse con SQL.
func Tables() []any {
	return []any{
		// catalogos
		&models.Departamento{},
		&models.Rol{},
		&models.Permiso{},
		&models.RolPermiso{},
		&models.Sector{},
		&models.TipoCaso{},
		&models.Institucion{},
		&models.Proposito{},
		&models.AvisoPrivacidad{},
		&models.AvisoProposito{},

		// identidad
		&models.Actor{},
		&models.Personal{},
		&models.Delegacion{},
		&models.MFAPersonal{},
		&models.SesionPersonal{},
		&models.ContextoAuth{},
		&models.ControlIntentos{},
		&models.RecuperacionMFA{},

		// seguridad
		&models.ClaveRegistro{},
		&models.Operacion{},

		// expedientes
		&models.Persona{},
		&models.IndiceCURP{},
		&models.Expediente{},
		&models.Episodio{},
		&models.HistorialEstado{},
		&models.Asignacion{},
		&models.Nota{},
		&models.TokenVinculacion{},
		&models.SolicitudVinculacion{},

		// consentimiento
		&models.Decision{},
		&models.DecisionProposito{},
		&models.Revocacion{},
		&models.RevocacionProposito{},

		// canalizacion
		&models.CanalizacionInterna{},
		&models.HistorialCanalizacion{},

		// auditoria
		&models.EventoAuditoria{},
		&models.AlertaAuditoria{},

		// ayuda
		&models.EventoLocalAyuda{},

		// capacitacion
		&models.Curso{},
		&models.Inscripcion{},
		&models.Visita{},
	}
}

// Migrate crea los esquemas, migra las tablas y crea las vistas.
//
// Es idempotente: AutoMigrate agrega columnas/índices faltantes y no borra datos;
// las vistas se recrean con DROP + CREATE.
//
// Limitaciones conocidas (ver README de migraciones):
//   - No crea CHECKs, RLS, triggers ni índices parciales/expresivos.
func Migrate(db *gorm.DB) error {
	for _, s := range schemas {
		if err := db.Exec("CREATE SCHEMA IF NOT EXISTS " + s).Error; err != nil {
			return fmt.Errorf("crear esquema %q: %w", s, err)
		}
	}

	if err := db.AutoMigrate(Tables()...); err != nil {
		return fmt.Errorf("automigrate: %w", err)
	}

	// AutoMigrate no crea índices funcionales. El login busca por lower(usuario),
	// así que la unicidad debe vivir en la base.
	if err := db.Exec(
		"CREATE UNIQUE INDEX IF NOT EXISTS ux_personal_usuario_lower ON identidad.personal (lower(usuario))",
	).Error; err != nil {
		return fmt.Errorf("índice único de usuario: %w", err)
	}

	if err := createViews(db); err != nil {
		return err
	}

	return nil
}
