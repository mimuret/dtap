/*
 * Copyright (c) 2022 Manabu Sonoda
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
package metrics

import (
	"fmt"

	"github.com/goccy/go-json"

	"github.com/mimuret/dnsutils/getter"
	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/promauto"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	_ = registry.RegisterFilterPlugin("metrics", Setup)
}

func Setup(bs json.RawMessage) (types.FilterPlugin, error) {
	s := &Metrics{}
	if err := json.Unmarshal(bs, s); err != nil {
		return nil, errors.Wrap(err, "failed to decode config")
	}
	if len(s.Rules) == 0 {
		return nil, errors.New("missing parameter Rules")

	}

	for i, rule := range s.Rules {
		if err := rule.Setup(); err != nil {
			return nil, errors.Wrapf(err, "invalid rules No %d", i)
		}
	}

	return s, nil
}

var _ types.FilterPlugin = &Metrics{}

// Specify if you want to get Prometheus
// label values from DNSTAP messages.
type DnstapLabel struct {
	// Label name
	Name string
	// getter func name
	// You can see getter names https://pkg.go.dev/github.com/mimuret/dnsutils/getter#DnstapGetterName.
	Attribute getter.DnstapGetterName
}

// Specify if you want to get Prometheus
// label values from DNS messages.
type DnsMsgLabel struct {
	// Label name
	Name string
	// You can see getter names https://pkg.go.dev/github.com/mimuret/dnsutils/getter#DnsMsgGetterName.
	Attribute getter.DnsMsgGetterName
}

type runLabel struct {
	name    string
	getFunc types.DnstapMessageGetFunc
}

// The Metrics plugin generates metrics.
// The generated metrics are output at the
// same time as the dtap metrics.
type MetricsRule struct {
	// Use the filter setting if there are messages
	// you want to exclude before calculating the
	// metrics.
	Filters plugin.FilterPlugins
	// Specify if you want to get Prometheus
	// label values from DNSTAP messages.
	DnstapLabels []DnstapLabel
	// Specify if you want to get Prometheus
	// label values from DNS messages.
	DnsMsgLabels []DnsMsgLabel
	// prometheus.CounterOpts
	// Set metrics name, constLabel, etc.
	CounterOps prometheus.CounterOpts

	runLabel []runLabel
	labels   []string
	counter  *prometheus.CounterVec
}

func (c *MetricsRule) Setup() error {
	labels := map[string]struct{}{}
	for i, label := range c.DnstapLabels {
		if label.Name == "" {
			return fmt.Errorf("DnstapLabel[%d] missing Name", i)
		}
		strFunc := getter.NewDnstapStrFunc(label.Attribute)
		if strFunc == nil {
			return fmt.Errorf("DnstapLabel[%d] unknown Attribute %s", i, label.Attribute)
		}
		if _, exist := labels[label.Name]; exist {
			return fmt.Errorf("DnstapLabel[%d] duplicate Name %s", i, label.Name)
		}
		labels[label.Name] = struct{}{}
		c.runLabel = append(c.runLabel, runLabel{
			name:    label.Name,
			getFunc: types.NewGetFuncFromDnstap(strFunc),
		})
		c.labels = append(c.labels, string(label.Name))
	}
	for i, label := range c.DnsMsgLabels {
		if label.Name == "" {
			return fmt.Errorf("DnsMsgLabel[%d] missing Name", i)
		}
		strFunc := getter.NewDnsMsgStrFunc(label.Attribute)
		if strFunc == nil {
			return fmt.Errorf("DnsMsgLabel[%d] unknown Attribute %s", i, label.Attribute)
		}
		if _, exist := labels[label.Name]; exist {
			return fmt.Errorf("DnsMsgLabel[%d] duplicate Name %s", i, label.Name)
		}
		labels[label.Name] = struct{}{}
		c.runLabel = append(c.runLabel, runLabel{
			name:    label.Name,
			getFunc: types.NewGetFuncFromDnsMsg(strFunc),
		})
		c.labels = append(c.labels, string(label.Name))
	}
	c.counter = prometheus.NewCounterVec(c.CounterOps, c.labels)
	if err := promauto.DefaultRegisterer().Register(c.counter); err != nil {
		return errors.Wrap(err, "failed to register metrics")
	}
	return nil
}

func (c *MetricsRule) GetLabels(dm *types.DnstapMessage) []string {
	labels := make([]string, len(c.runLabel))
	for i, label := range c.runLabel {
		labels[i] = label.getFunc(dm)
	}
	return labels
}

type Metrics struct {
	plugin.PluginCommon
	Rules []*MetricsRule
}

func (f *Metrics) Filter(dm *types.DnstapMessage) *types.DnstapMessage {
	for _, rule := range f.Rules {
		if rule.Filters.Filter(dm) == nil {
			continue
		}
		rule.counter.WithLabelValues(rule.GetLabels(dm)...).Inc()
	}
	return dm
}
