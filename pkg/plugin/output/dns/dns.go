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
	"time"

	json "github.com/goccy/go-json"
	"github.com/miekg/dns"
	"github.com/mimuret/dnsutils/dig"
	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/output"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
)

func init() {
	_ = registry.RegisterOutputPlugin("dns", setup)
}

type ExchangeContextInterface interface {
	ExchangeContext(ctx context.Context, m *dns.Msg, a string) (*dns.Msg, time.Duration, error)
}

func setup(bs json.RawMessage) (types.OutputPlugin, error) {
	s := &DNS{
		Protocol: DNSProtocolUDP,
		Timeout:  time.Second,
	}
	if err := json.Unmarshal(bs, s); err != nil {
		return nil, errors.Wrap(err, "failed to decode config")
	}

	if _, _, err := net.SplitHostPort(s.Host); err != nil {
		return nil, fmt.Errorf("invalid Host %s, %w", s.Host, err)
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

	cl *dig.Dig
	oc *types.OutputContext
}

func (o *DNS) SetOutputContext(oc *types.OutputContext) {
	o.oc = oc
}

func (o *DNS) Open() error {
	return nil
}

func (o *DNS) Write(dm *types.DnstapMessage) error {
	msg := dm.GetMessage()
	// skip response
	if msg.Response {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), o.Timeout)
	defer cancel()
	_, err := o.cl.ExchangeContext(ctx, msg)

	return err
}

func (o *DNS) Close() {
}
