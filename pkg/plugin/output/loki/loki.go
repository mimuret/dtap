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

	json "github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"

	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/types"

	"github.com/mimuret/dtap/v2/pkg/plugin/output"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"

	"github.com/go-kit/log"
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

	OutputFilters types.OutputFilters

	client.Config
	client.Client

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
	f.Client, err = client.New(f.metrics, f.Config, f.MaxStream, f.MaxLineSize, f.MaxLineSizeTruncate, &loggerWrapper{f.oc.Logger.Sugar()})
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
	entry := api.Entry{
		Labels: ToLabelSet(dm),
		Entry: logproto.Entry{
			Line: string(jsonRaw),
		},
	}
	select {
	case f.Client.Chan() <- entry:
	default:

	}
	return nil
}

func (f *Loki) Close() {
	f.Client.Stop()
}

func ToLabelSet(dm *types.DnstapMessage) model.LabelSet {
	set := make(model.LabelSet)
	for k, v := range dm.Labels {
		set[model.LabelName(k)] = model.LabelValue(v)
	}
	return set
}
