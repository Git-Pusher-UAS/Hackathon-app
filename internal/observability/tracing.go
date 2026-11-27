package observability

import (
	"context"
	"os"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

func InitTracing(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		otel.SetTracerProvider(trace.NewNoopTracerProvider())
		return func(context.Context) error { return nil }, nil
	}
	exp, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}
	res, _ := resource.Merge(resource.Default(), resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName(serviceName)))
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp), sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return tp.Shutdown, nil
}

func TracingMiddleware(serviceName string) fiber.Handler {
	tr := otel.Tracer(serviceName)
	return func(c *fiber.Ctx) error {
		ctx, span := tr.Start(c.UserContext(), c.Method()+" "+c.Route().Path)
		span.SetAttributes(attribute.String("http.method", c.Method()), attribute.String("http.path", c.Path()), attribute.String("request.id", string(c.Response().Header.Peek("X-Request-Id"))))
		c.SetUserContext(ctx)
		err := c.Next()
		span.SetAttributes(attribute.Int("http.status_code", c.Response().StatusCode()))
		span.End()
		return err
	}
}
