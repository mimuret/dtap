package oteltrace

import (
	"context"
	"fmt"
	"net"
	"net/url"

	"github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/output"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

func init() {
	_ = registry.RegisterOutputPlugin("otel-log", Setup)
}

func Setup(bs json.RawMessage) (types.OutputPlugin, error) {
	s := &OtelLog{
		ResourceAttributes: map[string]string{},
	}
	if err := json.Unmarshal(bs, s); err != nil {
		return nil, errors.Wrap(err, "failed to decode config")
	}
	if s.LoggerName == "" {
		s.LoggerName = "dtap"
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
type OtelLog struct {
	plugin.PluginCommon
	OutputFilters  types.OutputFilters
	AttributeNames []string

	// service name
	LoggerName         string
	ResourceAttributes map[string]string

	// OTLP Config
	OTLP *OTLPConfig
	// OTLPHTTP Config
	OTLPHTTP *OTLPHTTPConfig

	*output.DnstapOutput
	lp     *sdklog.LoggerProvider
	logger otellog.Logger
	oc     *types.OutputContext
}

func (f *OtelLog) SetOutputContext(oc *types.OutputContext) {
	f.oc = oc
}

func (f *OtelLog) Open() error {
	var (
		err      error
		exporter sdklog.Exporter
	)
	if f.OTLP != nil {
		exporter, err = GetOTLPExporter(context.Background(), f.OTLP.Endpoint, f.OTLP.Insecure, f.OTLP.Headers)
	} else if f.OTLPHTTP != nil {
		exporter, err = GetOTLPHTTPExporter(context.Background(), f.OTLPHTTP.Endpoint, f.OTLPHTTP.Insecure, f.OTLPHTTP.Headers)
	} else {
		exporter, err = GetSTDOUTExporter()
	}
	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}
	f.lp = sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(exporter)),
		sdklog.WithResource(GetResource(f.ResourceAttributes)),
	)
	f.logger = f.lp.Logger(f.LoggerName)

	return nil
}

func (f *OtelLog) Write(dm *types.DnstapMessage) error {
	attributes, err := f.GetAttributes(dm)
	if err != nil {
		return err
	}
	bs, err := dm.ConvertV1JSONWithFilter(f.OutputFilters)
	if err != nil {
		return err
	}
	record := otellog.Record{}
	record.SetBody(otellog.StringValue(string(bs)))
	record.SetEventName(dm.GetDnstap().Message.GetType().String())
	record.AddAttributes(attributes...)
	f.logger.Emit(context.Background(), otellog.Record{})
	return nil
}

func (f *OtelLog) Close() {
	f.lp.Shutdown(context.Background())
}

// Retrieve the attribute specified by AttributeNames from DNSTAP.
func (f *OtelLog) GetAttributes(dm *types.DnstapMessage) ([]otellog.KeyValue, error) {
	kv, err := dm.ConvertV1MapString()
	if err != nil {
		return nil, err
	}
	res := make([]otellog.KeyValue, 0, len(f.AttributeNames))
	for _, k := range f.AttributeNames {
		if v, ok := dm.Labels[k]; ok {
			res = append(res, otellog.KeyValue{Key: k, Value: otellog.StringValue(v)})
		}
		if v, ok := kv[k]; ok {
			switch v := v.(type) {
			case string:
				res = append(res, otellog.KeyValue{Key: k, Value: otellog.StringValue(v)})
			case int64:
				res = append(res, otellog.KeyValue{Key: k, Value: otellog.Int64Value(v)})
			case int:
				res = append(res, otellog.KeyValue{Key: k, Value: otellog.Int64Value(int64(v))})
			case int32:
				res = append(res, otellog.KeyValue{Key: k, Value: otellog.Int64Value(int64(v))})
			}
		}
	}
	return res, nil
}
