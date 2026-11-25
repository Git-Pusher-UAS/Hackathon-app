package models

import (
	"time"

	"github.com/google/uuid"
)

// Proyecciones de solo lectura (vistas SQL). No son tablas: no se migran con AutoMigrate.
// NOTA: el modelo relacional no define las columnas de estas vistas; las que
// aparecen aquí son una propuesta razonable. Ajústalas a tu DDL real.

// expedientes.v_timeline_autorizado
type TimelineItem struct {
	ItemID         uuid.UUID  `gorm:"column:item_id"`
	Tipo           string     `gorm:"column:tipo"` // nota | cambio_estado | canalizacion
	ExpedienteID   uuid.UUID  `gorm:"column:expediente_id"`
	DepartamentoID *uuid.UUID `gorm:"column:departamento_id"`
	ActorID        uuid.UUID  `gorm:"column:actor_id"`
	Resumen        string     `gorm:"column:resumen"` // estado o etiqueta; nunca texto de la nota en claro
	OcurridoEn     time.Time  `gorm:"column:ocurrido_en"`
	CreadoEn       time.Time  `gorm:"column:creado_en"`
}

func (TimelineItem) TableName() string { return "expedientes.v_timeline_autorizado" }

// consentimiento.v_estado_proposito: última decisión y revocación por persona y finalidad.
type EstadoProposito struct {
	PersonaID   uuid.UUID              `gorm:"column:persona_id"`
	PropositoID uuid.UUID              `gorm:"column:proposito_id"`
	DecisionID  uuid.UUID              `gorm:"column:decision_id"`
	Decision    DecisionConsentimiento `gorm:"column:decision"`
	Revocado    bool                   `gorm:"column:revocado"`
	Vigente     bool                   `gorm:"column:vigente"` // aceptado y sin revocación
	DecididoEn  time.Time              `gorm:"column:decidido_en"`
	RevocadoEn  *time.Time             `gorm:"column:revocado_en"`
}

func (EstadoProposito) TableName() string { return "consentimiento.v_estado_proposito" }

// analitica.v_resumen_mvp: agregados con supresión k=5. Un *int nil = celda suprimida (NULL).
type ResumenFila struct {
	SectorID   *uuid.UUID `gorm:"column:sector_id"`
	TipoCasoID *uuid.UUID `gorm:"column:tipo_caso_id"`
	Periodo    time.Time  `gorm:"column:periodo"`
	Personas   *int       `gorm:"column:personas"`
	Episodios  *int       `gorm:"column:episodios"`
}

func (ResumenFila) TableName() string { return "analitica.v_resumen_mvp" }

// analitica.v_mapa_mvp: nivel y conteo publicable por sector, sin puntos domiciliarios.
type MapaCelda struct {
	SectorID uuid.UUID `gorm:"column:sector_id"`
	Nivel    *string   `gorm:"column:nivel"`
	Conteo   *int      `gorm:"column:conteo"`
}

func (MapaCelda) TableName() string { return "analitica.v_mapa_mvp" }
