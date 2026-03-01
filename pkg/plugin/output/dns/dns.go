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
	"errors"
	"fmt"
	"math"
	"net"
	"runtime"
	"time"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/miekg/dns"
	"github.com/mimuret/dnsutils"
	"github.com/mimuret/dnsutils/dig"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/plugin/output"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/mimuret/dtap/v3/pkg/promauto"
	"github.com/mimuret/dtap/v3/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/semaphore"
)

const PLUGIN_NAME = "dns"

func init() {
	_ = registry.RegisterOutputPlugin(PLUGIN_NAME, Setup)
}

type ExchangeContextInterface interface {
	ExchangeContext(ctx context.Context, m *dns.Msg, a string) (*dns.Msg, time.Duration, error)
}

func Setup(cfg *config.OutputBlock) (types.OutputPlugin, error) {
	p := &DNS{
		OutputBlock: *cfg,
		Protocol:    DNSProtocolUDP,
		Timeout:     time.Second,
		WorkerNum:   4,
	}
	diags := gohcl.DecodeBody(cfg.Body, nil, p)
	if diags.HasErrors() {
		return nil, plugin.PluginError(p, "failed to setup dn plugin: %w", errors.Join(diags.Errs()...))
	}

	if _, _, err := net.SplitHostPort(p.Host); err != nil {
		return nil, plugin.PluginError(p, "host is an invalid address: %s", p.Host)
	}

	if p.WorkerNum == 0 {
		p.WorkerNum = uint(runtime.NumCPU())
	}

	switch p.Protocol {
	case DNSProtocolUDP, DNSProtocolTCP, DNSProtocolTLS:
	case DNSProtocolHTTPS, DNSProtocolHTTPSGet, DNSProtocolHTTPSPost:
	default:
		return nil, plugin.PluginError(p, "invalid protocol: %s", p.Protocol)
	}
	p.cl = dig.NewDig()
	p.cl.Client = &dns.Client{
		Net:     string(p.Protocol),
		Timeout: p.Timeout,
	}
	op := &dig.OptionTarget{Target: p.Host}
	if err := op.Option(p.cl); err != nil {
		return nil, fmt.Errorf("failed to set dig option: %w", err)
	}
	p.outCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "output_dns",
		Name:        "queries_total",
		ConstLabels: prometheus.Labels{"plugin": p.GetFullName()},
	})
	p.errCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "output_dns",
		Name:        "request_errors_total",
		ConstLabels: prometheus.Labels{"plugin": p.GetFullName()},
	})
	p.rcodeCoutner = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "output_dns",
		Name:        "responses_by_rcode_total",
		ConstLabels: prometheus.Labels{"plugin": p.GetFullName()},
	}, []string{"rcodes"})

	p.DnstapOutput = output.NewDnstapOutput(p, 0)
	return p, nil
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
	config.OutputBlock
	*output.DnstapOutput

	// Host is the DNS server address and port .
	Host string `hcl:"host"`

	// Transport Protocol
	// Supported protocols are "udp", "tcp", "tcp-tls", "https", "https-get", and "https-post".
	// Default is "udp".
	Protocol DNSProtocol `hcl:"protocol,optional"`

	// Timeout setting (duration)
	Timeout time.Duration `hcl:"timeout,optional"`

	// WorkerNum is the number of workers that process messages.
	WorkerNum uint `hcl:"worker_num,optional"`

	// UseResponseQuestion is a flag to use the question in the response message.
	UseResponseQuestion bool `hcl:"use_response_question,optional"`

	sem *semaphore.Weighted
	cl  *dig.Dig

	outCounter   prometheus.Counter
	errCounter   prometheus.Counter
	rcodeCoutner *prometheus.CounterVec
}

func (o *DNS) Open(context.Context) error {
	o.sem = semaphore.NewWeighted(int64(o.WorkerNum))
	return nil
}

func (o *DNS) Write(ctx context.Context, dm *types.DnstapMessage) error {
	if err := o.sem.Acquire(context.Background(), 1); err != nil {
		return err
	}
	go func(dm *types.DnstapMessage) {
		_ = o.write(dm)
		o.sem.Release(1)
	}(dm)
	return nil
}

func (o *DNS) write(dm *types.DnstapMessage) error {
	msg := dm.GetMessage()
	// UseResponseQuestion flag to use the question in the response message
	if msg.Response && !o.UseResponseQuestion {
		return nil
	}
	if len(msg.Question) == 0 {
		return nil
	}
	req := &dns.Msg{}
	req.SetQuestion(msg.Question[0].Name, msg.Question[0].Qtype)
	req.RecursionDesired = msg.RecursionDesired
	req.CheckingDisabled = msg.CheckingDisabled
	for _, rr := range msg.Extra {
		if rr.Header().Rrtype == dns.TypeOPT {
			edns0, ok := rr.(*dns.OPT)
			if !ok {
				break
			}
			if edns0.Do() {
				req.SetEdns0(edns0.UDPSize(), true)
			}
			break
		}
	}
	o.outCounter.Inc()
	res, err := o.cl.Exchange(req)
	if err != nil {
		o.errCounter.Inc()
		return err
	}
	rcodeStr := dnsutils.ConvertNumberToString(dns.RcodeToString, "RCODE", res.Rcode)
	o.rcodeCoutner.WithLabelValues(rcodeStr).Inc()
	return nil
}

func (o *DNS) Close(context.Context) {
}

func (p *DNS) MaxConcurrent() uint {
	return math.MaxUint32
}
