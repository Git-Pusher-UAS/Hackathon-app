package gormrepo

import (
	"fmt"

	"gorm.io/gorm"
)

// views se crean después de AutoMigrate porque dependen de las tablas base.
// Se usa DROP + CREATE (no CREATE OR REPLACE) para permitir cambiar columnas
// mientras el esquema todavía está en construcción.
var views = []struct {
	name string
	sql  string
}{
	{
		name: "expedientes.v_timeline_autorizado",
		sql: `
CREATE OR REPLACE VIEW expedientes.v_timeline_autorizado AS
SELECT
    n.id                  AS item_id,
    'nota'::varchar(20)   AS tipo,
    n.expediente_id,
    n.departamento_id,
    n.autor_id            AS actor_id,
    'nota'::text          AS resumen,
    n.ocurrido_en,
    n.creado_en
FROM expedientes.notas n
UNION ALL
SELECT
    h.id,
    'cambio_estado'::varchar(20),
    h.expediente_id,
    NULL::uuid,
    h.actor_id,
    h.estado_nuevo::text,
    h.creado_en,
    h.creado_en
FROM expedientes.historial_estados h
UNION ALL
SELECT
    c.id,
    'canalizacion'::varchar(20),
    c.expediente_id,
    c.destino_id,
    c.emitida_por,
    c.estado::text,
    c.creado_en,
    c.creado_en
FROM canalizacion.internas c;`,
	},
	{
		name: "consentimiento.v_estado_proposito",
		sql: `
CREATE OR REPLACE VIEW consentimiento.v_estado_proposito AS
WITH ultima AS (
    SELECT DISTINCT ON (d.persona_id, dp.proposito_id)
        d.persona_id,
        dp.proposito_id,
        d.id            AS decision_id,
        d.decision,
        d.capturado_en  AS decidido_en,
        dp.id           AS decision_proposito_id
    FROM consentimiento.decisiones d
    JOIN consentimiento.decision_propositos dp ON dp.decision_id = d.id
    ORDER BY d.persona_id, dp.proposito_id, d.orden_persona DESC, d.id DESC
),
rev AS (
    SELECT u.persona_id, u.proposito_id, max(r.ocurrido_en) AS revocado_en
    FROM ultima u
    JOIN consentimiento.revocacion_propositos rp
      ON rp.decision_proposito_id = u.decision_proposito_id
    JOIN consentimiento.revocaciones r
      ON r.id = rp.revocacion_id
    GROUP BY u.persona_id, u.proposito_id
)
SELECT
    u.persona_id,
    u.proposito_id,
    u.decision_id,
    u.decision,
    (rev.revocado_en IS NOT NULL)                         AS revocado,
    (u.decision = 'aceptado' AND rev.revocado_en IS NULL) AS vigente,
    u.decidido_en,
    rev.revocado_en
FROM ultima u
LEFT JOIN rev USING (persona_id, proposito_id);`,
	},
	{
		name: "analitica.v_resumen_mvp",
		sql: `
CREATE OR REPLACE VIEW analitica.v_resumen_mvp AS
SELECT
    p.sector_id,
    ep.tipo_caso_id,
    date_trunc('month', ep.abierto_en) AS periodo,
    CASE WHEN count(DISTINCT p.id)  >= 5 THEN count(DISTINCT p.id)::int  END AS personas,
    CASE WHEN count(DISTINCT ep.id) >= 5 THEN count(DISTINCT ep.id)::int END AS episodios
FROM expedientes.personas p
JOIN expedientes.expedientes e ON e.persona_id = p.id
JOIN expedientes.episodios ep  ON ep.expediente_id = e.id
GROUP BY p.sector_id, ep.tipo_caso_id, date_trunc('month', ep.abierto_en);`,
	},
	{
		name: "analitica.v_mapa_mvp",
		sql: `
CREATE OR REPLACE VIEW analitica.v_mapa_mvp AS
SELECT
    p.sector_id,
    ep.tipo_caso_id,
    date_trunc('month', ep.abierto_en) AS periodo,
    CASE WHEN count(DISTINCT ep.id) >= 5 THEN
        CASE
            WHEN count(DISTINCT ep.id) >= 20 THEN 'alto'
            WHEN count(DISTINCT ep.id) >= 10 THEN 'medio'
            ELSE 'bajo'
        END
    END AS nivel,
    CASE WHEN count(DISTINCT ep.id) >= 5 THEN count(DISTINCT ep.id)::int END AS conteo
FROM expedientes.personas p
JOIN expedientes.expedientes e ON e.persona_id = p.id
JOIN expedientes.episodios ep  ON ep.expediente_id = e.id
GROUP BY p.sector_id, ep.tipo_caso_id, date_trunc('month', ep.abierto_en);`,
	},
}

func createViews(db *gorm.DB) error {
	for _, v := range views {
		if err := db.Exec("DROP VIEW IF EXISTS " + v.name + " CASCADE").Error; err != nil {
			return fmt.Errorf("drop view %q: %w", v.name, err)
		}
		if err := db.Exec(v.sql).Error; err != nil {
			return fmt.Errorf("crear vista %q: %w", v.name, err)
		}
	}
	return nil
}

//se hizo asi pq no se podria usar views.go con el orm
