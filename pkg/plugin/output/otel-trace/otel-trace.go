package oteltrace

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"

	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/output"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func init() {
	_ = registry.RegisterOutputPlugin("otel-trace", Setup)
}

func Setup(bs json.RawMessage) (types.OutputPlugin, error) {
	s := &OtelTrace{
		ServiceName:        "dtap",
		SampleRate:         1.0,
		ResourceAttributes: map[string]string{},
	}
	if err := json.Unmarshal(bs, s); err != nil {
		return nil, errors.Wrap(err, "failed to decode config")
	}
	if s.ServiceName == "" {
		s.ServiceName = "dtap"
	}
	if s.OTLP != nil {
		if err := s.OTLP.Validate(); err != nil {
			return nil, errors.Wrap(err, "OTLP config error")
		}
	}
	if s.OTLPHTTP != nil {
		if err := s.OTLPHTTP.Validate(); err != nil {
			return nil, errors.Wrap(err, "OTLPHTTP config error")
		}
	}
	if s.OTLP == nil && s.OTLPHTTP == nil {
		return nil, errors.New("Either OTLP or OTLPHTTP must be set.")
	}
	if s.OTLP != nil && s.OTLPHTTP != nil {
		return nil, errors.New("Only one of OTLP or OTLPHTTP can be set.")
	}
	s.DnstapOutput = output.NewDnstapOutput(s, s.MaxRetry)
	return s, nil
}

type OTLPConfig struct {
	Endpoint string
	Insecure bool
	Headers  map[string]string
}

func (c *OTLPConfig) Validate() error {
	_, _, err := net.SplitHostPort(c.Endpoint)
	if err != nil {
		return fmt.Errorf("invalid OTLP endpoint: %w", err)
	}
	return nil
}

type OTLPHTTPConfig struct {
	Endpoint string
	Insecure bool
	Headers  map[string]string
}

func (c *OTLPHTTPConfig) Validate() error {
	if c.Endpoint == "" {
		return errors.New("invalid OTLPHTTP endpoint")
	}
	_, err := url.Parse(c.Endpoint)
	if err != nil {
		return fmt.Errorf("invalid OTLPHTTP endpoint: %w", err)
	}
	return nil
}

// The otel-trace plugin outputs messages to the OpenTelemetry collector.
type OtelTrace struct {
	plugin.PluginCommon
	OutputFilters types.OutputFilters

	// service name
	ServiceName        string
	SampleRate         float64
	ResourceAttributes map[string]string

	// OTLP Config
	OTLP *OTLPConfig
	// OTLPHTTP Config
	OTLPHTTP *OTLPHTTPConfig

	*output.DnstapOutput
	tp *sdktrace.TracerProvider
	oc *types.OutputContext
}

func (f *OtelTrace) SetOutputContext(oc *types.OutputContext) {
	f.oc = oc
}

func (f *OtelTrace) Open() error {
	var (
		err      error
		exporter *otlptrace.Exporter
	)
	if f.OTLP != nil {
		exporter, err = GetOTLPExporter(context.Background(), f.OTLP.Endpoint, f.OTLP.Insecure, f.OTLP.Headers)
	}
	if f.OTLPHTTP != nil {
		exporter, err = GetOTLPHTTPExporter(context.Background(), f.OTLPHTTP.Endpoint, f.OTLPHTTP.Insecure, f.OTLPHTTP.Headers)
	}
	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	f.tp = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(f.SampleRate)),
		sdktrace.WithResource(GetResource(f.ResourceAttributes)),
	)
	return nil
}

func (f *OtelTrace) Write(dm *types.DnstapMessage) error {
	attributes, err := f.GetAttributes(dm)
	if err != nil {
		return err
	}
	_, span := f.tp.Tracer("github.com/mimuret/dtap/v2/pkg/plugin/output/otel-trace").Start(
		context.Background(),
		dm.GetDnstap().Message.GetType().String())
	defer span.End()
	span.SetAttributes(attributes...)
	return nil
}

func (f *OtelTrace) Close() {
	f.tp.Shutdown(context.Background())
}

func (f *OtelTrace) GetAttributes(dm *types.DnstapMessage) ([]attribute.KeyValue, error) {
	kv, err := dm.ConvertV1MapStringWithFilter(f.OutputFilters)
	if err != nil {
		return nil, err
	}
	res := make([]attribute.KeyValue, 0, len(kv))
	for k, v := range kv {
		switch v := v.(type) {
		case string:
			res = append(res, attribute.KeyValue{Key: attribute.Key(k), Value: attribute.StringValue(v)})
		case int64:
			res = append(res, attribute.KeyValue{Key: attribute.Key(k), Value: attribute.Int64Value(v)})
		case int:
			res = append(res, attribute.KeyValue{Key: attribute.Key(k), Value: attribute.Int64Value(int64(v))})
		case int32:
			res = append(res, attribute.KeyValue{Key: attribute.Key(k), Value: attribute.Int64Value(int64(v))})
		}
	}
	return res, nil
}
