package oteltrace

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/url"

	"errors"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/plugin/output"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/mimuret/dtap/v3/pkg/types"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

const PLUGIN_NAME = "otel-log"

func init() {
	_ = registry.RegisterOutputPlugin(PLUGIN_NAME, Setup)
}

func Setup(cfg *config.OutputBlock) (types.OutputPlugin, error) {
	s := &OtelLog{
		OutputBlock:        *cfg,
		ResourceAttributes: map[string]string{},
		OutputFilters:      &types.OutputFilters{},
	}
	diags := gohcl.DecodeBody(cfg.Body, nil, s)
	if diags.HasErrors() {
		return nil, plugin.PluginError(s, "failed to setup otel-log plugin: %w", errors.Join(diags.Errs()...))
	}
	if s.LoggerName == "" {
		s.LoggerName = "dtap"
	}
	if s.OTLP != nil {
		if err := s.OTLP.Validate(); err != nil {
			return nil, plugin.PluginError(s, "OTLP config error: %w", err)
		}
	}
	if s.OTLPHTTP != nil {
		if err := s.OTLPHTTP.Validate(); err != nil {
			return nil, plugin.PluginError(s, "OTLPHTTP config error: %w", err)
		}
	}
	s.DnstapOutput = output.NewDnstapOutput(s, s.MaxRetry)
	return s, nil
}

type OTLPConfig struct {
	Endpoint string            `hcl:"endpoint"`
	Insecure bool              `hcl:"insecure,optional"`
	Headers  map[string]string `hcl:"headers,optional"`
}

func (c *OTLPConfig) Validate() error {
	_, _, err := net.SplitHostPort(c.Endpoint)
	if err != nil {
		return fmt.Errorf("invalid OTLP endpoint: %w", err)
	}
	return nil
}

type OTLPHTTPConfig struct {
	Endpoint string            `hcl:"endpoint"`
	Insecure bool              `hcl:"insecure,optional"`
	Headers  map[string]string `hcl:"headers,optional"`
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
	config.OutputBlock

	// OutputFilters is the list of filters to apply to the output.
	OutputFilters *types.OutputFilters `hcl:"output_filters,block"`

	// AttributeNames is the list of attribute names to be included in the log record.
	AttributeNames []string `hcl:"attribute_names,optional"`

	// service name
	LoggerName string `hcl:"logger_name,optional"`

	// Resource attributes
	ResourceAttributes map[string]string `hcl:"resource_attributes,optional"`

	// OTLP Config
	OTLP *OTLPConfig `hcl:"otlp,block,optional"`
	// OTLPHTTP Config
	OTLPHTTP *OTLPHTTPConfig `hcl:"otlp_http,block,optional"`

	// MaxRetry is the maximum number of retries to open the OTLP connection.
	MaxRetry uint `hcl:"max_retry,optional"`

	*output.DnstapOutput
	lp     *sdklog.LoggerProvider
	logger otellog.Logger
}

func (f *OtelLog) Open(context.Context) error {
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
		return plugin.PluginError(f, "failed to create trace exporter: %w", err)
	}
	f.lp = sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(exporter)),
		sdklog.WithResource(GetResource(f.ResourceAttributes)),
	)
	f.logger = f.lp.Logger(f.LoggerName)

	return nil
}

func (f *OtelLog) Write(ctx context.Context, dm *types.DnstapMessage) error {
	attributes, err := f.GetAttributes(dm)
	if err != nil {
		return err
	}
	bs, err := dm.ConvertV1JSONWithFilter(*f.OutputFilters)
	if err != nil {
		return err
	}
	record := otellog.Record{}
	record.SetBody(otellog.StringValue(string(bs)))
	record.SetEventName(dm.GetDnstap().Message.GetType().String())
	record.AddAttributes(attributes...)
	f.logger.Emit(ctx, otellog.Record{})
	return nil
}

func (f *OtelLog) Close(ctx context.Context) {
	f.lp.Shutdown(ctx)
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

func (p *OtelLog) MaxConcurrent() uint {
	return math.MaxUint32
}
