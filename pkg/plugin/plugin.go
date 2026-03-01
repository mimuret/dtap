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
package plugin

import (
	"github.com/mimuret/dtap/v3/pkg/buffer"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/promauto"
	"github.com/mimuret/dtap/v3/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
)

const defaultBufferSize = 10000

func NewBuffer(namespace, subsystem string, bufferSize uint, fullName string) types.Buffer {
	inCounter := promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   namespace,
		Subsystem:   subsystem,
		Name:        "recv_frames_total",
		Help:        "The total number of received frames",
		ConstLabels: prometheus.Labels{"plugin": fullName},
	})

	lostCounter := promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   namespace,
		Subsystem:   subsystem,
		Name:        "lost_frames_total",
		Help:        "The total number of lost frames from buffer",
		ConstLabels: prometheus.Labels{"plugin": fullName},
	})
	if bufferSize == 0 {
		bufferSize = defaultBufferSize
	}
	return buffer.NewRingBuffer(bufferSize, inCounter, lostCounter)
}

func NewBufferFromOutput(p *config.OutputBlock) types.Buffer {
	return NewBuffer("dtap", "output", p.GetBufferSize(), p.GetFullName())
}

func NewBufferFromFilterPlugin(p *config.FilterBlock) types.Buffer {
	return NewBuffer("dtap", "filter", p.GetBufferSize(), p.GetFullName())
}
