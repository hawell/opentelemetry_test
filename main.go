package main

import (
	"context"
	"go.opentelemetry.io/otel/baggage"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	exporter, err := otlp.NewExporter([]otlp.ExporterOption{otlp.WithInsecure(), otlp.WithAddress(os.Getenv("OTEL_AGENT_ENDPOINT"))}...)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %v", err)
	}
	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(bsp))
	defer func() { _ = tp.Shutdown(context.Background()) }()

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.Baggage{})

	tracer := otel.Tracer("ex.com/basic")
	ctx := context.Background()
	ctx = baggage.ContextWithValues(ctx, label.String("fooKey", "foo1"), label.String("barKey", "bar1"))

	err = func(ctx context.Context) error {
		var span trace.Span
		ctx, span = tracer.Start(ctx, "operation")
		defer span.End()

		span.AddEvent("Nice operation!", trace.WithAttributes(label.Int("bogons", 100)))
		span.SetAttributes(label.String("anotherKey", "yes"))

		return func(ctx context.Context) error {
			var span trace.Span
			ctx, span = tracer.Start(ctx, "Sub operation...")
			defer span.End()

			span.SetAttributes(label.String("lemonsKey", "five"))
			span.AddEvent("Sub span event")

			return nil
		}(ctx)
	}(ctx)
	if err != nil {
		panic(err)
	}

}