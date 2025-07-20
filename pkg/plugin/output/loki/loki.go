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
	"sync"
	"time"

	json "github.com/goccy/go-json"
	"github.com/grafana/dskit/backoff"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	prometheusconfig "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"

	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/mimuret/dtap/v2/pkg/utils"

	"github.com/mimuret/dtap/v2/pkg/plugin/output"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/loki/v3/clients/pkg/promtail/api"
	"github.com/grafana/loki/v3/clients/pkg/promtail/client"
	"github.com/grafana/loki/v3/pkg/logproto"
)

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
	_ = registry.RegisterOutputPlugin("loki", Setup)
}

func Setup(bs json.RawMessage) (types.OutputPlugin, error) {
	s := &Loki{
		MaxStream:   0,
		MaxLineSize: 4096,
	}
	if err := json.Unmarshal(bs, s); err != nil {
		return nil, errors.Wrap(err, "failed to decode config")
	}
	s.metrics = client.NewMetrics(prometheus.DefaultRegisterer)
	s.DnstapOutput = output.NewDnstapOutput(s, s.MaxRetry)

	urlvalue := flagext.URLValue{}
	if err := urlvalue.Set(s.URL); err != nil {
		return nil, errors.Wrap(err, "failed to set URL")
	}
	s.lokiConfig = client.Config{
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
	s.lokiConfig.Client = s.ClientConfig
	return s, nil
}

var _ output.OutputHandler = &Loki{}
var _ types.OutputPlugin = &Loki{}

// The loki plugin outputs messages to the loki server.
type Loki struct {
	plugin.PluginCommon
	sync.Mutex

	MaxStream           int
	MaxLineSize         int
	MaxLineSizeTruncate bool

	// loki body filter
	OutputFilters types.OutputFilters
	// loki labels from DNSTAP value
	DNSTAPLabels []string
	// loki labels
	Labels map[string]string
	// loki url
	URL          string
	ClientConfig prometheusconfig.HTTPClientConfig

	lokiConfig client.Config
	lokiClient client.Client

	metrics *client.Metrics
	*output.DnstapOutput
	oc *types.OutputContext
}

func (f *Loki) SetOutputContext(oc *types.OutputContext) {
	f.oc = oc
}

func (f *Loki) Open() error {
	var (
		err error
	)
	f.lokiClient, err = client.New(f.metrics, f.lokiConfig, f.MaxStream, f.MaxLineSize, f.MaxLineSizeTruncate, &loggerWrapper{f.oc.Logger.Sugar()})
	if err != nil {
		return errors.Wrap(err, "failed to create loki client")
	}
	return nil
}

func (f *Loki) Write(dm *types.DnstapMessage) error {
	jsonRaw, err := dm.ConvertV1JSONWithFilter(f.OutputFilters)
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

func (f *Loki) Close() {
	f.lokiClient.Stop()
}

func (f *Loki) ToLabelSet(dm *types.DnstapMessage) model.LabelSet {
	set := make(model.LabelSet)
	set[model.LabelName("level")] = "info"
	kv, err := dm.ConvertV1MapString()
	if err != nil {
		return set
	}
	for _, k := range f.DNSTAPLabels {
		if v, ok := dm.Labels[k]; ok {
			set[model.LabelName(k)] = model.LabelValue(v)
		}
		if v, ok := kv[k]; ok {
			if v, err := utils.ToString(v); err == nil {
				set[model.LabelName(k)] = model.LabelValue(v)
			}
		}
	}
	for k, v := range f.Labels {
		set[model.LabelName(k)] = model.LabelValue(v)
	}
	return set
}
