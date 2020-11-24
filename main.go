package main

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	"go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"log"
	"math/rand"
	"os"
	"time"
)

func main() {
	exporter, err := otlp.NewExporter([]otlp.ExporterOption{otlp.WithInsecure(), otlp.WithAddress(os.Getenv("OTEL_AGENT_ENDPOINT"))}...)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %v", err)
	}
	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(bsp))
	defer func() { _ = tp.Shutdown(context.Background()) }()

	pusher := push.New(
		basic.New(
			simple.NewWithExactDistribution(),
			exporter,
		),
		exporter,
	)
	pusher.Start()
	defer pusher.Stop()

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(pusher.MeterProvider())
	otel.SetTextMapPropagator(propagation.Baggage{})

	tracer := otel.Tracer("ex.com/basic")
	meter := otel.Meter("ex.com/basic")

	ctx := context.Background()
	ctx = baggage.ContextWithValues(ctx, label.String("fooKey", "foo1"), label.String("barKey", "bar1"))

	c1 := metric.Must(meter).NewInt64Counter("c1")
	c1.Add(ctx, int64(100), []label.KeyValue{label.String("A", "B")}...)

	for {
		err = func(ctx context.Context) error {
			var span trace.Span
			ctx, span = tracer.Start(ctx, "operation")
			defer func() {
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(1000)))
				span.End()
			}()

			span.AddEvent("Nice operation!", trace.WithAttributes(label.Int("bogons", 100)))
			span.SetAttributes(label.String("anotherKey", "yes"))

			return func(ctx context.Context) error {
				var span trace.Span
				ctx, span = tracer.Start(ctx, "Sub operation...")
				defer func() {
					time.Sleep(time.Millisecond * time.Duration(rand.Intn(1000)))
					span.End()
				}()

				span.SetAttributes(label.String("lemonsKey", "five"))
				span.AddEvent("Sub span event")

				return nil
			}(ctx)
		}(ctx)
		if err != nil {
			panic(err)
		}
	}
}