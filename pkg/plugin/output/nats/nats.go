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

package nats

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"time"

	"errors"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/promauto"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/types"

	"github.com/mimuret/dtap/v3/pkg/plugin/output"
	"github.com/mimuret/dtap/v3/pkg/plugin/pub"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/nats-io/nats.go"
)

const DefaultMaxPayloadSize = 1024 * 1024
const DnstapFstrmControlHeaderSize = 42
const DnstapFstrmMsgHeaderSize = 4

const PLUGIN_NAME = "nats"

func init() {
	_ = registry.RegisterOutputPlugin(PLUGIN_NAME, Setup)
}

func Setup(cfg *config.OutputBlock) (types.OutputPlugin, error) {
	p := &Nats{
		OutputBlock: *cfg,
		MaxSize:     DefaultMaxPayloadSize,
		Format:      pub.FormatDtapFrame,
		Interval:    time.Second,
	}
	diags := gohcl.DecodeBody(cfg.Body, nil, p)
	if diags.HasErrors() {
		return nil, plugin.PluginError(p, "failed to setup loki plugin: %w", errors.Join(diags.Errs()...))
	}
	if len(p.Hosts) == 0 {
		return nil, plugin.PluginError(p, "missing parameter hosts")
	}
	if p.Subject == "" {
		return nil, plugin.PluginError(p, "missing parameter subject")
	}
	if p.Token == "" && p.User != "" && p.Password == "" {
		return nil, plugin.PluginError(p, "missing parameter password")
	}
	if p.Token == "" && p.User == "" && p.Password != "" {
		return nil, plugin.PluginError(p, "missing parameter user")
	}
	if p.Interval == 0 {
		p.Interval = time.Second
	}
	p.publisher = pub.NewPublisher(p.Format, p.MaxSize, p.Interval, p)
	if p.publisher == nil {
		return nil, plugin.PluginError(p, "failed to create publisher for format %s", p.Format)
	}
	if p.TLSConfig != nil {
		tlsConfig, err := p.TLSConfig.CryptoTLSConfig()
		if err != nil {
			return nil, plugin.PluginError(p, "failed to create TLS config: %w", err)
		}
		p.tlsConfig = tlsConfig
	}

	p.DnstapOutput = output.NewDnstapOutput(p, p.MaxRetry)

	p.openErr = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "output_nats",
		Name:        "open_errors_total",
		ConstLabels: prometheus.Labels{"plugin": p.GetFullName()},
	})
	p.publishCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "output_nats",
		Name:        "publishes_total",
		ConstLabels: prometheus.Labels{"plugin": p.GetFullName()},
	})
	p.publishErrCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "output_nats",
		Name:        "publish_failed_total",
		ConstLabels: prometheus.Labels{"plugin": p.GetFullName()},
	})
	p.writeMessageCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "output_nats",
		Name:        "write_messages_total",
		ConstLabels: prometheus.Labels{"plugin": p.GetFullName()},
	})
	p.writeMessageErrCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "output_nats",
		Name:        "write_errors_total",
		ConstLabels: prometheus.Labels{"plugin": p.GetFullName()},
	})
	p.publishDurationSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace:   "dtap",
		Subsystem:   "output_nats",
		Name:        "publish_duration_seconds",
		ConstLabels: prometheus.Labels{"plugin": p.GetFullName()},
	})
	return p, nil
}

var _ output.OutputHandler = &Nats{}
var _ types.OutputPlugin = &Nats{}
var _ pub.PublisherHandler = &Nats{}

// The nats plugin outputs messages to the nats server.
// DTAPFrame or DNSTAP messages are taken from the Output
// Buffer and pushed together to the nats server.
// The timing of the push is when the size of the nats
// message is exceeded or when the number of seconds specified
// by IntervalSec elapses.
type Nats struct {
	config.OutputBlock

	*output.DnstapOutput

	// Hosts is the URL of the nats servers. Must not be empty.
	Hosts []string `hcl:"hosts"`
	// Nats subject. Must not be empty.
	Subject string `hcl:"subject"`

	// Nats user, If a token is given, it is not used.
	User string `hcl:"user,optional"`

	// Nats password, If a token is given, it is not used.
	Password string `hcl:"password,optional"`

	// Nats token
	Token string `hcl:"token,optional"`

	// Secure indicates whether to use a secure connection (TLS).
	Secure bool `hcl:"secure,optional"`

	// TLSConfig is the TLS configuration for the NATS connection.
	TLSConfig *config.TLSClientConfig `hcl:"tls_config,block"`

	// Max nats message size. Default is 1KByte.
	MaxSize int `hcl:"max_size,optional"`

	// Interval to flush nats messages.
	Interval time.Duration `hcl:"interval,optional"`

	// Nats message format.
	Format pub.Format `hcl:"format,optional"`

	// MaxRetry is the maximum number of retries to connect nats server.
	MaxRetry uint `hcl:"max_retry,optional"`

	conn      *nats.Conn
	tlsConfig *tls.Config

	publisher pub.Publisher

	openErr prometheus.Counter

	publishCounter         prometheus.Counter
	publishErrCounter      prometheus.Counter
	publishDurationSeconds prometheus.Histogram
	writeMessageCounter    prometheus.Counter
	writeMessageErrCounter prometheus.Counter
}

func (p *Nats) Open(ctx context.Context) error {
	var err error

	cfg := nats.GetDefaultOptions()
	cfg.Servers = p.Hosts
	if p.Token != "" {
		cfg.Token = p.Token
	} else if p.User != "" {
		cfg.User = p.User
		cfg.Password = p.Password
	}
	cfg.Secure = p.Secure
	if p.Secure && p.tlsConfig != nil {
		cfg.TLSConfig = p.tlsConfig
	}
	p.conn, err = cfg.Connect()
	if err != nil {
		p.openErr.Inc()
		return plugin.PluginError(p, "failed to create nats producer: %w", err)
	}
	p.publisher.Start(ctx)
	return nil
}

func (p *Nats) Write(ctx context.Context, dm *types.DnstapMessage) error {
	p.writeMessageCounter.Inc()
	if err := p.publisher.Write(ctx, dm); err != nil {
		ctxzap.Debug(ctx, "failed to write", zap.Any("message", dm), zap.Error(err))
		p.writeMessageErrCounter.Inc()
	}
	return nil
}

func (p *Nats) Publish(ctx context.Context, data []byte) error {
	start := time.Now().Unix()
	err := p.conn.Publish(p.Subject, data)
	p.publishDurationSeconds.Observe(float64(time.Now().Unix() - start))
	p.publishCounter.Inc()
	if err != nil {
		p.publishErrCounter.Inc()
		return fmt.Errorf("publish error: %w", err)
	}
	return nil
}

func (p *Nats) Close(ctx context.Context) {
	p.publisher.Close(ctx)
	p.conn.Close()
}

func (p *Nats) MaxConcurrent() uint {
	return math.MaxUint32
}
