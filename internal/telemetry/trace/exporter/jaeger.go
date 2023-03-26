package exporter

import (
	"fmt"

	"go.opentelemetry.io/otel/exporters/jaeger"
)

func NewJaeger(endpoint string) (*jaeger.Exporter, error) {
	fmt.Println("create otlp grpc client")
	traceExp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
	if err != nil {
		return nil, err
	}
	fmt.Println("Done")
	return traceExp, nil
}
