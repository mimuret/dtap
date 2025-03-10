package oteltrace

import (
	"context"
	"fmt"
	"net"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
)

func GetOTLPExporter(ctx context.Context, hostAndPort string, insecure bool, headers map[string]string, options ...otlptracegrpc.Option) (*otlptrace.Exporter, error) {
	_, _, err := net.SplitHostPort(hostAndPort)
	if err != nil {
		return nil, fmt.Errorf("invalit OTLP endpoint: %w", err)
	}
	options = append(options,
		otlptracegrpc.WithEndpoint(hostAndPort),
		otlptracegrpc.WithHeaders(headers),
	)
	if insecure {
		options = append(options, otlptracegrpc.WithInsecure())
	}

	return otlptracegrpc.New(ctx, options...)
}

func GetOTLPHTTPExporter(ctx context.Context, url string, insecure bool, headers map[string]string, options ...otlptracehttp.Option) (*otlptrace.Exporter, error) {
	options = append(options,
		otlptracehttp.WithEndpoint(url),
		otlptracehttp.WithHeaders(headers),
	)
	if insecure {
		options = append(options, otlptracehttp.WithInsecure())
	}
	return otlptracehttp.NewUnstarted(options...), nil
}
