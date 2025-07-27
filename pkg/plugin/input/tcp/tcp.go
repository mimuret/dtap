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
package tcp

import (
	"context"
	"net"
	"strconv"

	"errors"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/plugin/input"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/mimuret/dtap/v3/pkg/types"
	"go.uber.org/zap"
)

const PLUGIN_NAME = "tcp"

func init() {
	_ = registry.RegisterInputPlugin(PLUGIN_NAME, Setup)
}

func Setup(cfg *config.InputBlock) (types.InputPlugin, error) {
	p := &TCPSocket{
		InputBlock: *cfg,
		Format:     input.FormatDNSTAP,
	}

	// Decode the HCL body into the TCPSocket struct.
	diags := gohcl.DecodeBody(cfg.Body, nil, p)
	if diags.HasErrors() {
		return nil, plugin.PluginError(p, "failed to setup TCPSocket plugin: %w", errors.Join(diags.Errs()...))
	}
	if net.ParseIP(p.Address) == nil {
		return nil, plugin.PluginError(p, "invalid address: %s", p.Address)
	}
	if p.Port == 0 {
		return nil, plugin.PluginError(p, "missing parameter Port")
	}
	p.is = input.NewInputServer(p, p.Format, nil)
	if p.is == nil {
		return nil, plugin.PluginError(p, "invalid format: %s", p.Format)
	}

	return p, nil
}

var _ types.InputPlugin = &TCPSocket{}

// The TCPSocket plugin get messages from the tcp socket.
// Example HCL configuration:
// ```hcl
//
//		input "tcp" "tcp_example" {
//		  address = "127.0.0.1"
//		  port    = 12345
//		  format  = "DNSTAP"
//		}
//		input "tcp" "tls_example" {
//		  address = "127.0.0.1"
//		  port    = 12345
//		  format  = "DNSTAP"
//		  tls_config {
//		    certificate = "/path/to/cert.pem"
//		    private_key  = "/path/to/key.pem"
//	   }
//		}
//
// ```

type TCPSocket struct {
	config.InputBlock

	// Format specifies the message format. Optional, defaults to "DNSTAP".
	Format string `hcl:"format,optional"`

	// Listen Address. Must not be empty.
	Address string `hcl:"address"`

	// Listen port. Must not be empty.
	Port uint16 `hcl:"port"`

	TLSConfig *config.TLSServerConfig `hcl:"tls_config,block"`

	is *input.InputServer
}

func (p *TCPSocket) listen(addr string) (net.Listener, error) {
	if p.TLSConfig == nil {
		return net.Listen("tcp", addr)
	}
	return p.TLSConfig.Listen(addr)
}

func (p *TCPSocket) Start(ctx context.Context, forwarder types.Forwarder) error {
	addr := net.JoinHostPort(p.Address, strconv.Itoa(int(p.Port)))
	ln, err := p.listen(addr)
	if err != nil {
		return plugin.PluginError(p, "failed to listen on %s: %w", addr, err)
	}
	go func() {
		<-ctx.Done()
		if err := ln.Close(); err != nil {
			ctxzap.Error(ctx, "failed to close TCPSocket", zap.Error(err))
		}
	}()
	return p.is.Serve(ctx, forwarder, ln)
}
