package exporter

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"google.golang.org/grpc"
)

func NewOTLP(endpoint string) *otlptrace.Exporter {
	ctx := context.Background()
	fmt.Println("create otlp grpc client")
	traceClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithDialOption(grpc.WithBlock()))

	fmt.Println("create otlp exporter")
	traceExp, err := otlptrace.New(ctx, traceClient)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to create the collector trace exporter")
	}
	fmt.Println("Done")
	return traceExp
}
