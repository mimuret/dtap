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
	"context"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

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
	Name        string
	Vec         *prometheus.CounterVec
	LabelKeys   []string
	LabelValues map[string]*DnstapPrometheusOutputMetricsValues
	Interval    int
	Expire      int
	CancelFunc  context.CancelFunc
}

func (d *DnstapPrometheusOutputMetrics) GetInterval() int {
	if d.Interval <= 0 {
		return 0
	}
	return d.Interval
}

func (d *DnstapPrometheusOutputMetrics) GetExpire() int {
	if d.Expire <= 0 {
		return 0
	}
	return d.Expire
}

type DnstapPrometheusOutputMetricsValues struct {
	Values     []string
	LastUpdate time.Time
}

func NewDnstapPrometheusOutputMetrics(counterConfig OutputPrometheusMetrics) *DnstapPrometheusOutputMetrics {
	return &DnstapPrometheusOutputMetrics{
		Name: counterConfig.GetName(),
		Vec: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: counterConfig.GetName(),
			Help: counterConfig.GetHelp(),
		}, counterConfig.GetLabels()),
		LabelKeys:   counterConfig.GetLabels(),
		LabelValues: map[string]*DnstapPrometheusOutputMetricsValues{},
		Expire:      counterConfig.GetExpireSec(),
		Interval:    counterConfig.GetExpireInterval(),
	}
}

func (d *DnstapPrometheusOutputMetrics) Inc(values []string) {
	d.Vec.WithLabelValues(values...).Inc()
	d.LabelValues[strings.Join(values, ",")] = &DnstapPrometheusOutputMetricsValues{
		Values:     values,
		LastUpdate: time.Now(),
	}
}

func (d *DnstapPrometheusOutputMetrics) Flush(ctx context.Context) {
	ticker := time.NewTicker(time.Second * time.Duration(d.Interval))
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for k, value := range d.LabelValues {
				if time.Now().Sub(value.LastUpdate) > time.Second*time.Duration(d.Expire) {
					d.Vec.DeleteLabelValues(value.Values...)
					delete(d.LabelValues, k)
				}
			}
		}
	}
}

func NewDnstapPrometheusOutput(config *OutputPrometheus, params *DnstapOutputParams) *DnstapOutput {
	p := &DnstapPrometheusOutput{
		config:  config,
		Metrics: []*DnstapPrometheusOutputMetrics{},
	}
	for _, counterConfig := range config.GetCounters() {
		p.Metrics = append(p.Metrics, NewDnstapPrometheusOutputMetrics(counterConfig))
	}
	params.Handler = p
	return NewDnstapOutput(params)
}

func (o *DnstapPrometheusOutput) open() error {
	for _, metrics := range o.Metrics {
		if metrics.GetInterval() > 0 && metrics.GetExpire() > 0 {
			ctx, cancelFunc := context.WithCancel(context.Background())
			metrics.CancelFunc = cancelFunc
			go metrics.Flush(ctx)
		}
	}
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
		field := e.Type().Field(i).Name
		value := e.Field(i).Interface()
		if value == nil {
			continue
		}
		switch v := value.(type) {
		case string:
			m[field] = v
		case uint32:
			m[field] = strconv.Itoa(int(v))
		case uint16:
			m[field] = strconv.Itoa(int(v))
		case bool:
			if v {
				m[field] = "1"
			} else {
				m[field] = "0"
			}
		case fmt.Stringer:
			m[field] = v.String()
		}
	}

	for _, counter := range o.Metrics {
		labelValues := make([]string, 0, len(counter.LabelKeys))
		for _, l := range counter.LabelKeys {
			if v, ok := m[l]; ok {
				labelValues = append(labelValues, v)
			} else {
				log.Warnf("can't get metrics: %v, %v", l, counter.Name)
			}
		}
		counter.Inc(labelValues)
	}
	return nil
}

func (o *DnstapPrometheusOutput) close() {
	for _, metrics := range o.Metrics {
		if metrics.CancelFunc != nil {
			metrics.CancelFunc()
		}
	}
}
