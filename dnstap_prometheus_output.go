/*
 * Copyright (c) 2019 Manabu Sonoda
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

package dtap

import (
	"fmt"
	"reflect"
	"strconv"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type DnstapPrometheusOutput struct {
	config  *OutputPrometheus
	Metrics []*DnstapPrometheusOutputMetrics
}

type DnstapPrometheusOutputMetrics struct {
	Vec    *prometheus.CounterVec
	Labels []string
}

func NewDtapCounterVec(opts prometheus.CounterOpts) *prometheus.CounterVec {
	s := DnstapFlatT{}
	t := reflect.TypeOf(s)
	var labels []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		j := field.Tag.Get("json")
		labels = append(labels, j)
	}
	return promauto.NewCounterVec(opts, labels)
}

func NewDnstapPrometheusOutput(config *OutputPrometheus, params *DnstapOutputParams) *DnstapOutput {
	p := &DnstapPrometheusOutput{
		config:  config,
		Metrics: []*DnstapPrometheusOutputMetrics{},
	}
	for _, counterConfig := range config.GetCounters() {
		counter := &DnstapPrometheusOutputMetrics{
			Vec: NewDtapCounterVec(prometheus.CounterOpts{
				Name: counterConfig.GetName(),
				Help: counterConfig.GetHelp(),
			}),
			Labels: counterConfig.GetLabels(),
		}
		p.Metrics = append(p.Metrics, counter)
	}
	params.Handler = p
	return NewDnstapOutput(params)
}

func (o *DnstapPrometheusOutput) open() error {
	return nil
}

func (o *DnstapPrometheusOutput) write(frame []byte) error {
	dt := dnstap.Dnstap{}
	if err := proto.Unmarshal(frame, &dt); err != nil {
		return err
	}
	data, err := FlatDnstap(&dt, &o.config.Flat)
	if err != nil {
		return err
	}
	e := reflect.ValueOf(data).Elem()
	m := make(map[string]string)
	for i := 0; i < e.NumField(); i++ {
		field := e.Type().Field(i).Tag.Get("json")
		value := e.Field(i).Interface()
		switch v := value.(type) {
		case string:
			m[field] = v
		case uint32:
			m[field] = strconv.Itoa(int(v))
		case uint16:
			m[field] = strconv.Itoa(int(v))
		case fmt.Stringer:
			m[field] = v.String()
		}
	}

	for _, counter := range o.Metrics {
		labels := map[string]string{}
		for _, l := range counter.Labels {
			if v, ok := m[l]; ok {
				labels[l] = v
			}
		}
		counter.Vec.With(labels).Inc()
	}
	return nil
}

func (o *DnstapPrometheusOutput) close() {
}
