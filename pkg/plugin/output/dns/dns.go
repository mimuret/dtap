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
package stdout

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"time"

	json "github.com/goccy/go-json"
	"github.com/miekg/dns"
	"github.com/mimuret/dnsutils"
	"github.com/mimuret/dnsutils/dig"
	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/output"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/promauto"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/semaphore"
)

func init() {
	_ = registry.RegisterOutputPlugin("dns", setup)
}

type ExchangeContextInterface interface {
	ExchangeContext(ctx context.Context, m *dns.Msg, a string) (*dns.Msg, time.Duration, error)
}

func setup(bs json.RawMessage) (types.OutputPlugin, error) {
	s := &DNS{
		Protocol:  DNSProtocolUDP,
		Timeout:   time.Second,
		WorkerNum: 4,
	}
	if err := json.Unmarshal(bs, s); err != nil {
		return nil, errors.Wrap(err, "failed to decode config")
	}

	if _, _, err := net.SplitHostPort(s.Host); err != nil {
		return nil, fmt.Errorf("invalid Host %s, %w", s.Host, err)
	}

	if s.WorkerNum == 0 {
		s.WorkerNum = uint(runtime.NumCPU())
	}

	switch s.Protocol {
	case DNSProtocolUDP, DNSProtocolTCP, DNSProtocolTLS:
	case DNSProtocolHTTPS, DNSProtocolHTTPSGet, DNSProtocolHTTPSPost:
	default:
		return nil, fmt.Errorf("invalid Protocol %s", s.Protocol)
	}
	s.cl = dig.NewDig()
	s.cl.Client = &dns.Client{
		Net: string(s.Protocol),
	}
	op := &dig.OptionTarget{Target: s.Host}
	if err := op.Option(s.cl); err != nil {
		return nil, fmt.Errorf("failed to set dig option: %w", err)
	}
	s.outCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "output_dns",
		Name:        "queries_total",
		ConstLabels: prometheus.Labels{"ID": s.GetID()},
	})
	s.errCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "output_dns",
		Name:        "request_errors_total",
		ConstLabels: prometheus.Labels{"ID": s.GetID()},
	})
	s.rcodeCoutner = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "output_dns",
		Name:        "responses_by_rcode_total",
		ConstLabels: prometheus.Labels{"ID": s.GetID()},
	}, []string{"rcodes"})

	s.DnstapOutput = output.NewDnstapOutput(s, 0)
	return s, nil
}

type DNSProtocol string

var (
	DNSProtocolUDP       DNSProtocol = "udp"
	DNSProtocolTCP       DNSProtocol = "tcp"
	DNSProtocolTLS       DNSProtocol = "tcp-tls"
	DNSProtocolHTTPS     DNSProtocol = "https"
	DNSProtocolHTTPSGet  DNSProtocol = "https-get"
	DNSProtocolHTTPSPost DNSProtocol = "https-post"
)

var _ types.OutputPlugin = &DNS{}

// This is an experimental implementation.
// The DNS plugin sends the same question as
// the message to the DNS server.
type DNS struct {
	plugin.PluginCommon
	*output.DnstapOutput

	// target DNS server
	Host string
	// Transport Protocol
	Protocol DNSProtocol
	// Timeout setting (duration)
	Timeout time.Duration
	// SendOnly flag
	WorkerNum uint

	sem *semaphore.Weighted
	cl  *dig.Dig
	oc  *types.OutputContext

	outCounter   prometheus.Counter
	errCounter   prometheus.Counter
	rcodeCoutner *prometheus.CounterVec
}

func (o *DNS) SetOutputContext(oc *types.OutputContext) {
	o.oc = oc
}

func (o *DNS) Open() error {
	o.sem = semaphore.NewWeighted(int64(o.WorkerNum))
	return nil
}

func (o *DNS) Write(dm *types.DnstapMessage) error {
	o.sem.Acquire(context.Background(), 1)
	go func(dm *types.DnstapMessage) {
		o.write(dm)
		o.sem.Release(1)
	}(dm)
	return nil
}

func (o *DNS) write(dm *types.DnstapMessage) error {
	msg := dm.GetMessage()
	// skip response
	if msg.Response {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), o.Timeout)
	defer cancel()
	o.outCounter.Inc()
	res, err := o.cl.ExchangeContext(ctx, msg)
	if err != nil {
		o.errCounter.Inc()
		return err
	}
	rcodeStr := dnsutils.ConvertNumberToString(dns.RcodeToString, "RCODE", res.Rcode)
	o.rcodeCoutner.WithLabelValues(rcodeStr).Inc()
	return nil
}

func (o *DNS) Close() {
}
