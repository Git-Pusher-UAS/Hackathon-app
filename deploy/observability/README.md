# Observabilidad

## Métricas
- Endpoint Prometheus: `GET /metrics`
- Middleware HTTP: `sai_http_requests_total`, `sai_http_request_duration_seconds`
- Seguridad/negocio: `sai_rate_limit_blocked_total`, `sai_otp_challenges_total`, `sai_otp_verifications_total`, `sai_contact_releases_total`, `sai_audit_append_errors_total`

## Trazas (OTel)
- Configurar `OTEL_EXPORTER_OTLP_ENDPOINT` para exportar trazas OTLP/HTTP.
- Sin la variable, el tracer queda en no-op.

## Logs
- Logs estructurados JSON en stdout con `request_id`, método, ruta, status, latencia.

## Alertas
- Ver `prometheus-alerts.yml`.

## Dashboard
- Importar `grafana-dashboard.json` en Grafana.
