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
	"io"
	"math"
	"net"
	"strconv"
	"time"

	"errors"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/plugin/output"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/mimuret/dtap/v3/pkg/types"
)

const PLUGIN_NAME = "tcp"

func init() {
	_ = registry.RegisterOutputPlugin(PLUGIN_NAME, Setup)
}

func Setup(cfg *config.OutputBlock) (types.OutputPlugin, error) {
	p := &TCP{
		OutputBlock: *cfg,
	}
	diags := gohcl.DecodeBody(cfg.Body, nil, p)
	if diags.HasErrors() {
		return nil, plugin.PluginError(p, "failed to setup tcp plugin: %w", errors.Join(diags.Errs()...))
	}
	p.DnstapOutput = output.NewDnstapOutput(output.NewDnstapFstrmSocketOutput(p, time.Second, nil), p.MaxRetry)
	return p, nil
}

var _ types.OutputPlugin = &TCP{}
var _ output.SocketOutput = &TCP{}

// The TCP plugin outputs messages to the tcp server.
// Eample configuration:
// ```hcl
//
//	output "tcp" "tcp_test" {
//	  host = "127.0.0.1"
//	  port = 12345
//	  max_retry = 3
//	}
//
//	output "tcp" "tcp_test_tls" {
//		 host = "127.0.0.1"
//		 port = 12346
//	  max_retry = 3
//	  tls_config {}
//	}
//
// ```
type TCP struct {
	config.OutputBlock
	*output.DnstapOutput

	// TCP server hostname. Must not be empty.
	Host string `hcl:"host"`

	// TCP server port number. Must not be empty.
	Port uint16 `hcl:"port"`

	// MaxRetry is the maximum number of retries to open the unix socket.
	MaxRetry uint `hcl:"max_retry,optional"`

	TLSConfig *config.TLSClientConfig `hcl:"tls_config,block"`
	w         net.Conn
}

func (p *TCP) NewConnect(context.Context) (io.Writer, error) {
	var err error
	target := net.JoinHostPort(p.Host, strconv.Itoa(int(p.Port)))
	if p.TLSConfig != nil {
		p.w, err = p.TLSConfig.Dial(target)
	} else {
		p.w, err = net.Dial("tcp", target)
	}
	if err != nil {
		return nil, plugin.PluginError(p, "failed to connect tcp socket, address: %s: err: %w", target, err)
	}
	return p.w, nil
}

func (p *TCP) Close(context.Context) {
	p.w.Close()
}

func (p *TCP) MaxConcurrent() uint {
	return math.MaxUint32
}
