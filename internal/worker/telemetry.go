package worker

import (
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	// metricExporter "audio_compression/internal/telemetry/metric/exporter"
	ttrace "audio_compression/internal/telemetry/trace"
	traceExporter "audio_compression/internal/telemetry/trace/exporter"
)

func (s *Worker) InitGlobalProvider(name, endpoint string) {
	// metricExp := metricExporter.NewOTLP(endpoint)
	// pusher, pusherCloseFn, err := metric.NewMeterProviderBuilder().
	// 	SetExporter(metricExp).
	// 	SetHistogramBoundaries([]float64{5, 10, 25, 50, 100, 200, 400, 800, 1000}).
	// 	Build()
	// if err != nil {
	// 	log.Fatal().Err(err).Msgf("failed initializing the meter provider")
	// }
	// s.metricProviderCloseFn = append(s.metricProviderCloseFn, pusherCloseFn)
	// global.SetMeterProvider(pusher)

	spanExporter, err := traceExporter.NewJaeger(endpoint)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed initializing the tracer exporter")
	}

	tracerProvider, tracerProviderCloseFn, err := ttrace.NewTraceProviderBuilder(name).
		SetExporter(spanExporter).
		Build()
	if err != nil {
		log.Fatal().Err(err).Msgf("failed initializing the tracer provider")
	}
	s.traceProviderCloseFn = append(s.traceProviderCloseFn, tracerProviderCloseFn)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	otel.SetTracerProvider(tracerProvider)
}
