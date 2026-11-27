package models

// ---- Vocabularios cerrados (CHECK en la BD) ----

type TipoActor string

const (
	ActorPersonal TipoActor = "personal"
	ActorSistema  TipoActor = "sistema"
)

type TipoSector string

const (
	SectorSector      TipoSector = "sector"
	SectorColonia     TipoSector = "colonia"
	SectorSindicatura TipoSector = "sindicatura"
)

type EstadoPersonal string

const (
	PersonalActivo     EstadoPersonal = "activo"
	PersonalSuspendido EstadoPersonal = "suspendido"
)

type EstadoMFA string

const (
	MFAPendiente  EstadoMFA = "pendiente"
	MFAActivo     EstadoMFA = "activo"
	MFAInvalidado EstadoMFA = "invalidado"
)

type FaseContexto string

const (
	FaseAnonimo      FaseContexto = "anonimo"
	FaseTOTP         FaseContexto = "totp"
	FaseEnrolamiento FaseContexto = "enrolamiento"
	// FaseActivacion es el código de un solo uso para elegir la contraseña.
	FaseActivacion FaseContexto = "activacion"
)

type EstadoExpediente string

const (
	ExpedienteAbierto       EstadoExpediente = "abierto"
	ExpedienteEnSeguimiento EstadoExpediente = "en_seguimiento"
	ExpedienteCerrado       EstadoExpediente = "cerrado"
)

type OrigenContacto string

const (
	OrigenPresencial         OrigenContacto = "presencial"
	OrigenInstitucion        OrigenContacto = "institucion"
	OrigenLineaAtencion      OrigenContacto = "linea_atencion"
	OrigenPatrullaGenero     OrigenContacto = "patrulla_genero"
	OrigenEmergenciaReferida OrigenContacto = "emergencia_referida"
)

type EstadoSolicitudVinculacion string

const (
	SolicitudPendiente EstadoSolicitudVinculacion = "pendiente"
	SolicitudAprobada  EstadoSolicitudVinculacion = "aprobada"
	SolicitudRechazada EstadoSolicitudVinculacion = "rechazada"
)

type DecisionConsentimiento string

const (
	DecisionAceptado  DecisionConsentimiento = "aceptado"
	DecisionRechazado DecisionConsentimiento = "rechazado"
)

type ModalidadConsentimiento string

const (
	ModalidadPresencial ModalidadConsentimiento = "presencial"
	ModalidadDigital    ModalidadConsentimiento = "digital"
)

type TipoCanalizacion string

const (
	CanalizacionInternaTipo TipoCanalizacion = "interna" // a otro departamento
	CanalizacionExternaTipo TipoCanalizacion = "externa" // a una institución de catalogos.instituciones
)

type EstadoCanalizacion string

const (
	CanalizacionPendiente  EstadoCanalizacion = "pendiente"
	CanalizacionAceptada   EstadoCanalizacion = "aceptada"
	CanalizacionRechazada  EstadoCanalizacion = "rechazada"
	CanalizacionCancelada  EstadoCanalizacion = "cancelada"
	CanalizacionCompletada EstadoCanalizacion = "completada"
)

type ResultadoAuditoria string

const (
	ResultadoExito          ResultadoAuditoria = "exito"
	ResultadoDenegado       ResultadoAuditoria = "denegado"
	ResultadoError          ResultadoAuditoria = "error"
	ResultadoIntentoEntrega ResultadoAuditoria = "intento_entrega"
)

type NivelAlerta string

const (
	AlertaMedia NivelAlerta = "media"
	AlertaAlta  NivelAlerta = "alta"
)

type EstadoAlerta string

const (
	AlertaAbierta  EstadoAlerta = "abierta"
	AlertaRevisada EstadoAlerta = "revisada"
)

type EstadoOperacion string

const (
	OperacionAplicada EstadoOperacion = "aplicada"
)

// AccionAyuda es el único valor permitido por el CHECK de ayuda.eventos_locales.
const AccionAyudaDialerOpened = "dialer_opened"
