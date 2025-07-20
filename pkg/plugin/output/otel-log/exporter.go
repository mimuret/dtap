package oteltrace

import (
	"context"
	"fmt"
	"net"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/sdk/log"
)

func GetOTLPExporter(ctx context.Context, hostAndPort string, insecure bool, headers map[string]string, options ...otlploggrpc.Option) (*otlploggrpc.Exporter, error) {
	_, _, err := net.SplitHostPort(hostAndPort)
	if err != nil {
		return nil, fmt.Errorf("invalit OTLP endpoint: %w", err)
	}
	options = append(options,
		otlploggrpc.WithEndpoint(hostAndPort),
		otlploggrpc.WithHeaders(headers),
	)
	if insecure {
		options = append(options, otlploggrpc.WithInsecure())
	}

	return otlploggrpc.New(ctx, options...)
}

func GetOTLPHTTPExporter(ctx context.Context, url string, insecure bool, headers map[string]string, options ...otlploghttp.Option) (*otlploghttp.Exporter, error) {
	options = append(options,
		otlploghttp.WithEndpoint(url),
		otlploghttp.WithHeaders(headers),
	)
	if insecure {
		options = append(options, otlploghttp.WithInsecure())
	}
	return otlploghttp.New(ctx, options...)
}

func GetSTDOUTExporter() (log.Exporter, error) {
	return stdoutlog.New()
}
