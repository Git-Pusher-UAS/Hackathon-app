package gormrepo

import (
	repository "backend-hackathon/internal/repository/repositories"

	"gorm.io/gorm"
)

// Compilación: el store debe cumplir todos los getters de repository.Repositories.
var _ repository.Repositories = (*store)(nil)

// store liga todos los repositorios a la misma conexión o transacción.
type store struct {
	db *gorm.DB
}

// NewRepositories construye el agregado de repositorios sobre una conexión o tx.
func NewRepositories(db *gorm.DB) repository.Repositories {
	return &store{db: db}
}

// ---- catalogos ----

func (s *store) Departamentos() repository.DepartamentoRepository {
	return &departamentoRepo{db: s.db}
}
func (s *store) Roles() repository.RolRepository { return &rolRepo{db: s.db} }
func (s *store) Permisos() repository.PermisoRepository {
	return &permisoRepo{db: s.db}
}
func (s *store) RolPermisos() repository.RolPermisoRepository {
	return &rolPermisoRepo{db: s.db}
}
func (s *store) Sectores() repository.SectorRepository { return &sectorRepo{db: s.db} }
func (s *store) TiposCaso() repository.TipoCasoRepository {
	return &tipoCasoRepo{db: s.db}
}
func (s *store) Instituciones() repository.InstitucionRepository {
	return &institucionRepo{db: s.db}
}
func (s *store) Propositos() repository.PropositoRepository {
	return &propositoRepo{db: s.db}
}
func (s *store) AvisosPrivacidad() repository.AvisoPrivacidadRepository {
	return &avisoPrivacidadRepo{db: s.db}
}
func (s *store) AvisoPropositos() repository.AvisoPropositoRepository {
	return &avisoPropositoRepo{db: s.db}
}

// ---- identidad ----

func (s *store) Actores() repository.ActorRepository { return &actorRepo{db: s.db} }
func (s *store) Personal() repository.PersonalRepository {
	return &personalRepo{db: s.db}
}
func (s *store) Delegaciones() repository.DelegacionRepository {
	return &delegacionRepo{db: s.db}
}
func (s *store) MFA() repository.MFAPersonalRepository { return &mfaRepo{db: s.db} }
func (s *store) Sesiones() repository.SesionPersonalRepository {
	return &sesionRepo{db: s.db}
}
func (s *store) ContextosAuth() repository.ContextoAuthRepository {
	return &contextoAuthRepo{db: s.db}
}
func (s *store) ControlIntentos() repository.ControlIntentosRepository {
	return &controlIntentosRepo{db: s.db}
}
func (s *store) RecuperacionesMFA() repository.RecuperacionMFARepository {
	return &recuperacionMFARepo{db: s.db}
}

// ---- seguridad ----

func (s *store) ClavesRegistro() repository.ClaveRegistroRepository {
	return &claveRegistroRepo{db: s.db}
}
func (s *store) Operaciones() repository.OperacionRepository {
	return &operacionRepo{db: s.db}
}

// ---- expedientes ----

func (s *store) Personas() repository.PersonaRepository { return &personaRepo{db: s.db} }
func (s *store) IndicesCURP() repository.IndiceCURPRepository {
	return &indiceCURPRepo{db: s.db}
}
func (s *store) Expedientes() repository.ExpedienteRepository {
	return &expedienteRepo{db: s.db}
}
func (s *store) Episodios() repository.EpisodioRepository {
	return &episodioRepo{db: s.db}
}
func (s *store) HistorialEstados() repository.HistorialEstadoRepository {
	return &historialEstadoRepo{db: s.db}
}
func (s *store) Asignaciones() repository.AsignacionRepository {
	return &asignacionRepo{db: s.db}
}
func (s *store) Notas() repository.NotaRepository { return &notaRepo{db: s.db} }
func (s *store) TokensVinculacion() repository.TokenVinculacionRepository {
	return &tokenVinculacionRepo{db: s.db}
}
func (s *store) SolicitudesVinculacion() repository.SolicitudVinculacionRepository {
	return &solicitudVinculacionRepo{db: s.db}
}
func (s *store) Timeline() repository.TimelineRepository { return &timelineRepo{db: s.db} }

// ---- consentimiento ----

func (s *store) Decisiones() repository.DecisionRepository {
	return &decisionRepo{db: s.db}
}
func (s *store) DecisionPropositos() repository.DecisionPropositoRepository {
	return &decisionPropositoRepo{db: s.db}
}
func (s *store) Revocaciones() repository.RevocacionRepository {
	return &revocacionRepo{db: s.db}
}
func (s *store) RevocacionPropositos() repository.RevocacionPropositoRepository {
	return &revocacionPropositoRepo{db: s.db}
}
func (s *store) EstadoPropositos() repository.EstadoPropositoRepository {
	return &estadoPropositoRepo{db: s.db}
}

// ---- canalizacion ----

func (s *store) Canalizaciones() repository.CanalizacionRepository {
	return &canalizacionRepo{db: s.db}
}
func (s *store) HistorialCanalizaciones() repository.HistorialCanalizacionRepository {
	return &historialCanalizacionRepo{db: s.db}
}

// ---- auditoria ----

func (s *store) EventosAuditoria() repository.EventoAuditoriaRepository {
	return &eventoAuditoriaRepo{db: s.db}
}
func (s *store) AlertasAuditoria() repository.AlertaAuditoriaRepository {
	return &alertaAuditoriaRepo{db: s.db}
}

// ---- ayuda ----

func (s *store) EventosAyuda() repository.EventoLocalAyudaRepository {
	return &eventoLocalAyudaRepo{db: s.db}
}

// ---- analitica (vistas) ----

func (s *store) Resumen() repository.ResumenRepository         { return &resumenRepo{db: s.db} }
func (s *store) Mapa() repository.MapaRepository               { return &mapaRepo{db: s.db} }
func (s *store) Indicadores() repository.IndicadoresRepository { return &indicadoresRepo{db: s.db} }

// ---- portal ----

func (s *store) PortalCuentas() repository.PortalCuentaRepository { return &portalCuentaRepo{db: s.db} }
func (s *store) PortalContactos() repository.PortalContactoRepository {
	return &portalContactoRepo{db: s.db}
}
func (s *store) PortalDesafios() repository.PortalDesafioRepository {
	return &portalDesafioRepo{db: s.db}
}
func (s *store) PortalSesiones() repository.PortalSesionRepository {
	return &portalSesionRepo{db: s.db}
}
func (s *store) PortalAlertas() repository.PortalAlertaRepository { return &portalAlertaRepo{db: s.db} }

// ---- capacitacion ----

func (s *store) Cursos() repository.CursoRepository              { return &cursoRepo{db: s.db} }
func (s *store) Inscripciones() repository.InscripcionRepository { return &inscripcionRepo{db: s.db} }
func (s *store) Visitas() repository.VisitaRepository            { return &visitaRepo{db: s.db} }
