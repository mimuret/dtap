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
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"sync"

	framestream "github.com/farsightsec/golang-framestream"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/hashicorp/hcl/v2/gohcl"
	"go.uber.org/zap"

	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/types"

	"github.com/mimuret/dtap/v3/pkg/plugin/input"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/nats-io/nats.go"
)

const PLUGIN_NAME = "nats"

func init() {
	_ = registry.RegisterInputPlugin(PLUGIN_NAME, Setup)
}

func Setup(cfg *config.InputBlock) (types.InputPlugin, error) {
	p := &Nats{
		InputBlock: *cfg,
		Format:     input.FormatDtapFrame,
		QueueLen:   64,
	}
	// Decode the HCL body into the nats struct.
	diags := gohcl.DecodeBody(cfg.Body, nil, p)
	if diags.HasErrors() {
		return nil, plugin.PluginError(p, "failed to setup nats plugin: %w", errors.Join(diags.Errs()...))
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
	p.is = input.NewInputServer(p, p.Format, &framestream.DecoderOptions{
		Bidirectional: false,
	})
	if p.is == nil {
		return nil, plugin.PluginError(p, "invalid format")
	}
	if p.TLSConfig != nil {
		tlsConfig, err := p.TLSConfig.CryptoTLSConfig()
		if err != nil {
			return nil, plugin.PluginError(p, "failed to create TLS config: %w", err)
		}
		p.tlsConfig = tlsConfig
	}
	return p, nil
}

var _ types.InputPlugin = &Nats{}

// The nats plugin retrieves messages from the nats server.
// It supports subscribing to a specific subject and queue.
// The plugin receives messages in the FSTRM format.
// The plugin can handle messages in various formats, including DNSTAP and DtapFrame.
// Authentication can be done using a token or user/password credentials.
// Example configuration:
// ```hcl
//
//	input "nats" "nats_example" {
//	  hosts = ["nats://localhost:4222"]
//	  subject = "example.subject"
//	  queue_name = "example_queue"
//	  queue_len = 64
//	  user = "username" // optional
//	  password = "password" // optional, if token is not set
//	  token = "your_token" // optional, if user and password are not set
//	  format = "DtapFrame" // optional, default is "DtapFrame"
//	}
//
// ```
type Nats struct {
	config.InputBlock

	// Nats payload format default is "DtapFrame".
	// Supported values are "DNSTAP", "DtapFrame",
	Format string `hcl:"format,optional"`

	// Hosts is the URL of the nats servers. Must not be empty.
	Hosts []string `hcl:"hosts"`

	// Nats subject. Must not be empty.
	Subject string `hcl:"subject"`

	// Nats queue name.
	QueueName string `hcl:"queue_name,optional"`

	// QueueLen is nats subscriber queue length.
	QueueLen int `hcl:"queue_len,optional"`

	// Nats user, If a token is given, it is not used.
	// If a user is given, password must be given.
	User string `hcl:"user,optional"`

	// Nats password, If a token is given, it is not used.
	Password string `hcl:"password,optional"`

	// Nats token
	// If a token is given, user and password are not used.
	Token string `hcl:"token,optional"`

	// Secure indicates whether to use a secure connection (TLS).
	Secure bool `hcl:"secure,optional"`

	// TLSConfig is the TLS configuration for the NATS connection.
	TLSConfig *config.TLSClientConfig `hcl:"tls_config,block"`

	is        *input.InputServer
	tlsConfig *tls.Config
}

func (p *Nats) Open() (*nats.Conn, error) {
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
	conn, err := cfg.Connect()
	if err != nil {
		return nil, plugin.PluginError(p, "failed to connect to NATS server: %w", err)
	}
	return conn, nil
}

func (p *Nats) Start(ctx context.Context, forwarder types.Forwarder) error {
	nc, err := p.Open()
	if err != nil {
		return plugin.PluginError(p, "failed to connect to NATS server: %w", err)
	}
	defer func() {
		_ = nc.Drain()
	}()
	ch := make(chan *nats.Msg, p.QueueLen)
	defer close(ch)

	sub, err := nc.ChanQueueSubscribe(p.Subject, p.QueueName, ch)
	if err != nil {
		return plugin.PluginError(p, "failed to subscribe: %w", err)
	}
	defer func() {
		_ = sub.Unsubscribe()
	}()
	wg := sync.WaitGroup{}
	defer wg.Wait()
	ctxzap.Debug(ctx, "start subscribe", zap.String("subject", p.Subject), zap.String("queue name", p.QueueName), zap.Int("queue len", p.QueueLen))
LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case msg := <-ch:
			wg.Add(1)
			go func(bs []byte) {
				defer wg.Done()
				buf := bytes.NewBuffer(bs)
				if err := p.is.Read(ctx, forwarder, buf); err != nil {
					ctxzap.Debug(ctx, "input error", zap.Error(err))
				}
			}(msg.Data)
		}
	}
	return nil
}
