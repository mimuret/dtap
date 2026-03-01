/*
 * Copyright (c) 2023 Manabu Sonoda
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package loki

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/grafana/dskit/backoff"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"

	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/types"

	"github.com/mimuret/dtap/v3/pkg/plugin/output"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/loki/v3/clients/pkg/promtail/api"
	"github.com/grafana/loki/v3/clients/pkg/promtail/client"
	"github.com/grafana/loki/v3/pkg/logproto"
)

const PLUGIN_NAME = "loki"

var _ log.Logger = &loggerWrapper{}

type loggerWrapper struct {
	*zap.SugaredLogger
}

func (l *loggerWrapper) Log(keyvals ...interface{}) error {
	l.Infow("", keyvals...)
	return nil
}

const DefaultMaxPayloadSize = 1024 * 1024
const DnstapFstrmControlHeaderSize = 42
const DnstapFstrmMsgHeaderSize = 4

func init() {
	_ = registry.RegisterOutputPlugin(PLUGIN_NAME, Setup)
}

func Setup(cfg *config.OutputBlock) (types.OutputPlugin, error) {
	p := &Loki{
		OutputBlock: *cfg,
	}
	s := &LokiConfig{
		MaxStream:     0,
		MaxLineSize:   4096,
		ConstLabels:   map[string]string{"level": "info"},
		OutputFilters: &types.OutputFilters{},
	}
	diags := gohcl.DecodeBody(cfg.Body, nil, s)
	if diags.HasErrors() {
		return nil, plugin.PluginError(p, "failed to setup loki plugin: %w", errors.Join(diags.Errs()...))
	}
	p.DnstapOutput = output.NewDnstapOutput(p, s.MaxRetry)

	urlvalue := flagext.URLValue{}
	if err := urlvalue.Set(s.URL); err != nil {
		return nil, fmt.Errorf("failed to set URL: %w", err)
	}

	p.outputFilters = *s.OutputFilters
	p.messageLabels = s.MessageLabels
	p.constLabels = s.ConstLabels
	p.maxStream = s.MaxStream
	p.maxLineSize = s.MaxLineSize
	p.maxLineSizeTruncate = s.MaxLineSizeTruncate

	p.lokiConfig = client.Config{
		URL: urlvalue,
		BackoffConfig: backoff.Config{
			MaxBackoff: client.MaxBackoff,
			MaxRetries: client.MaxRetries,
			MinBackoff: client.MinBackoff,
		},
		BatchSize: client.BatchSize,
		BatchWait: client.BatchWait,
		Timeout:   client.Timeout,
	}

	// Openが複数呼ばれると、registryのメトリクスが重複して登録されるため、ここで初期化する。
	p.metrics = client.NewMetrics(prometheus.DefaultRegisterer)

	return p, nil
}

var _ output.OutputHandler = &Loki{}
var _ types.OutputPlugin = &Loki{}

type LokiConfig struct {
	// MaxStream is the maximum number of streams to be sent to loki.
	MaxStream int `hcl:"max_stream,optional"`
	// MaxLineSize is the maximum size of a line to be sent to loki.
	MaxLineSize int `hcl:"max_line_size,optional"`
	// MaxLineSizeTruncate indicates whether to truncate the line if it exceeds MaxLineSize.
	MaxLineSizeTruncate bool `hcl:"max_line_size_truncate,optional"`
	// loki body filter
	OutputFilters *types.OutputFilters `hcl:"output_filters,block"`
	// loki labels from DNSTAP value
	MessageLabels []string `hcl:"message_labels,optional"`
	// loki labels
	ConstLabels map[string]string `hcl:"const_labels,optional"`
	// loki url
	URL string `hcl:"url"`
	// MaxRetry is the maximum number of retries to connect nats server.
	MaxRetry uint `hcl:"max_retry,optional"`
}

// The loki plugin outputs messages to the loki server.
// Example configuration:
// ```hcl
// output "loki" "example" {
//   url = "http://localhost:3100/loki/api/v1/push"
//   max_stream = 1000
//   max_line_size = 4096
//   max_line_size_truncate = true
//   output_filters {
//     include = ["type", "query", "response_code"]
//     exclude = ["query_time", "response_time"]
//   }
//   message_labels = ["type", "query", "response_code"]
//   const_labels = {
//     "app": "dtap",
//     "env": "production"
//   }
//   max_retry = 3
// }
// ```

type Loki struct {
	config.OutputBlock

	// OutputFilters is the list of filters to apply to the output.
	outputFilters types.OutputFilters

	// MessageLabels Labels are the labels to be added to all log entries.
	messageLabels []string

	// const_labels are the constant labels to be added to all log entries.
	constLabels map[string]string

	maxStream           int
	maxLineSize         int
	maxLineSizeTruncate bool
	lokiConfig          client.Config

	lokiClient client.Client

	metrics *client.Metrics
	*output.DnstapOutput
}

func (f *Loki) Open(ctx context.Context) error {
	var (
		err error
	)
	f.lokiClient, err = client.New(f.metrics, f.lokiConfig, f.maxStream, f.maxLineSize, f.maxLineSizeTruncate, &loggerWrapper{ctxzap.Extract(ctx).Sugar()})
	if err != nil {
		return fmt.Errorf("failed to create loki client: %w", err)
	}
	return nil
}

func (f *Loki) Write(ctx context.Context, dm *types.DnstapMessage) error {
	jsonRaw, err := dm.ConvertV1JSONWithFilter(f.outputFilters)
	if err != nil {
		return err
	}
	timestamp := dm.GetTimestamp()
	if timestamp == nil {
		now := time.Now()
		timestamp = &now
	}
	entry := api.Entry{
		Labels: f.ToLabelSet(dm),
		Entry: logproto.Entry{
			Timestamp: *timestamp,
			Line:      string(jsonRaw),
		},
	}
	f.lokiClient.Chan() <- entry
	return nil
}

func (f *Loki) Close(ctx context.Context) {
	f.lokiClient.Stop()
}

func (f *Loki) ToLabelSet(dm *types.DnstapMessage) model.LabelSet {
	set := make(model.LabelSet)
	for k, v := range f.constLabels {
		set[model.LabelName(k)] = model.LabelValue(v)
	}
	labels := make(map[string]string, len(f.messageLabels))
	if err := dm.SetOuputAttributes(labels, f.messageLabels); err != nil {
		return nil
	}
	for k, v := range labels {
		set[model.LabelName(k)] = model.LabelValue(v)
	}
	return set
}

func (p *Loki) MaxConcurrent() uint {
	return math.MaxUint32
}
