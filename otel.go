package main

import (
	"context"
	"errors"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

// setupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func setupOTelSDK(ctx context.Context) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Set up trace provider.
	tracerProvider, err := newTraceProvider()
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Set up meter provider.
	meterProvider, err := newMeterProvider()
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	// Set up logger provider.
	loggerProvider, err := newLoggerProvider()
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	global.SetLoggerProvider(loggerProvider)
	return
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTraceProvider() (*trace.TracerProvider, error) {
	traceExporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithEndpoint("localhost:5081"),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithHeaders(
			map[string]string{
				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDNjhVQXhzUUE1YnFpb2sz",
				"organization":  "default",
				"stream-name":   "default",
			}),
	)

	if err != nil {
		return nil, err
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter,
			// Default is 5s. Set to 1s for demonstrative purposes.
			trace.WithBatchTimeout(time.Second)),
	)
	return traceProvider, nil
}

func newMeterProvider() (*metric.MeterProvider, error) {
	metricExporter, err := otlpmetricgrpc.New(context.Background(),
		otlpmetricgrpc.WithEndpoint("localhost:5081"),
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithHeaders(map[string]string{
			"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDNjhVQXhzUUE1YnFpb2sz",
			"organization":  "default",
			"stream-name":   "default",
		}),
	)
	if err != nil {
		return nil, err
	}
	r, err := resource.New(
		context.Background(),
		resource.WithAttributes(semconv.ServiceNameKey.String("my-service")),
		resource.WithProcessPID(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithTelemetrySDK(),
		resource.WithProcess(),
	)

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(r),
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			// Default is 1m. Set to 3s for demonstrative purposes.
			metric.WithInterval(3*time.Second))),
	)
	return meterProvider, nil
}

func newLoggerProvider() (*log.LoggerProvider, error) {
	logExporter, err := otlploggrpc.New(context.Background(),
		otlploggrpc.WithEndpoint("localhost:5081"),
		otlploggrpc.WithHeaders(
			map[string]string{
				"Authorization": "Basic cm9vdEBleGFtcGxlLmNvbTpDNjhVQXhzUUE1YnFpb2sz",
				"organization":  "default",
				"stream-name":   "default",
			},
		),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
	)
	return loggerProvider, nil
}
