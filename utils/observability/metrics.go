package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"go.opentelemetry.io/otel"
)

// InitMetrics sets up OpenTelemetry metrics with a Prometheus exporter
// and returns an http.Handler for the /metrics endpoint.
func InitMetrics(serviceName string) (http.Handler, error) {
	exporter, err := promexporter.New()
	if err != nil {
		return nil, err
	}

	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	otel.SetMeterProvider(mp)

	return promhttp.Handler(), nil
}
