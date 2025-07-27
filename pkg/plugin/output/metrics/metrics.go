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
	"context"

	"errors"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/mimuret/dtap/v3/pkg/promauto"
	"github.com/mimuret/dtap/v3/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
)

const PLUGIN_NAME = "metrics"

func init() {
	_ = registry.RegisterOutputPlugin(PLUGIN_NAME, Setup)
}
func Setup(cfg *config.OutputBlock) (types.OutputPlugin, error) {
	p := &Metrics{
		OutputBlock: *cfg,
	}
	diags := gohcl.DecodeBody(cfg.Body, nil, p)
	if diags.HasErrors() {
		return nil, plugin.PluginError(p, "failed to setup metrics plugin: %w", errors.Join(diags.Errs()...))
	}
	if len(p.MessageLabels) == 0 {
		p.counter = promauto.NewCounter(prometheus.CounterOpts{
			Name: p.Name,
			Help: p.Description,
		})
	}
	if len(p.MessageLabels) > 0 {
		p.counterVec = promauto.NewCounterVec(prometheus.CounterOpts{
			Name: p.Name,
			Help: p.Description,
		}, p.MessageLabels)
	}

	return p, nil
}

var _ types.OutputPlugin = &Metrics{}

// Filter and Ouput processes the DnstapMessage and increments the counter
// based on the labels defined in the MetricsRule.
// Example Configuration:
// ```hcl
//
//	filter "nop" "metrics" {
//	  forward_to = [
//	     "output.metrics.custom_queries_total_by_qtype",
//	     "output.metrics.custom_queries_total_by_qtype",
//	  ]
//	}
//
//	output "metrics" "custom_queries_total_by_qtype" {
//	  labels = ["qtype"]
//	}
//
//	output "metrics" "custom_queries_total_by_faimly_and_protocol" {
//	  labels = ["socket_family", "socket_protocol"]
//	}
type Metrics struct {
	config.OutputBlock

	// key names of  types.ConvertV1MapString.
	MessageLabels []string `hcl:"message_labels,optional"`

	// Description is the description of the metrics.
	Description string `hcl:"description,optional"`

	counterVec *prometheus.CounterVec
	counter    prometheus.Counter
}

func (p *Metrics) Inc(dm *types.DnstapMessage) {
	if p.counter != nil {
		p.counter.Inc()
		return
	}
	if dm == nil {
		return
	}
	labels := make(map[string]string, len(p.MessageLabels))
	err := dm.SetOuputAttributes(labels, p.MessageLabels)
	if err != nil {
		return
	}
	p.counterVec.With(labels).Inc()
}

func (f *Metrics) Start(ctx context.Context, r types.Reader) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case dm, ok := <-r.Read():
			if !ok {
				return nil
			}
			if dm == nil {
				continue
			}
			f.Inc(dm)
		}
	}
}

func (p *Metrics) MaxConcurrent() uint {
	return uint(1)
}
