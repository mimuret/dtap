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
package unix

import (
	"context"
	"io"
	"math"
	"net"
	"time"

	"errors"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/plugin/output"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/mimuret/dtap/v3/pkg/types"
)

const PLUGIN_NAME = "unix"

func init() {
	_ = registry.RegisterOutputPlugin(PLUGIN_NAME, Setup)
}

func Setup(cfg *config.OutputBlock) (types.OutputPlugin, error) {
	p := &Unix{
		OutputBlock: *cfg,
	}
	diags := gohcl.DecodeBody(cfg.Body, nil, p)
	if diags.HasErrors() {
		return nil, plugin.PluginError(p, "failed to setup unix plugin: %w", errors.Join(diags.Errs()...))
	}
	p.DnstapOutput = output.NewDnstapOutput(output.NewDnstapFstrmSocketOutput(p, time.Second, nil), p.MaxRetry)
	return p, nil
}

var _ types.OutputPlugin = &Unix{}
var _ output.SocketOutput = &Unix{}

// The unix plugin outputs messages to unix socket.
// Example configuration:
// ```hcl
//
//	output "unix" "unix_test" {
//	  path = "/var/run/dtap.sock"
//	  max_retry = 3
//	}
//
// ```
type Unix struct {
	config.OutputBlock
	*output.DnstapOutput

	// unix socket path
	Path string `hcl:"path"`

	// MaxRetry is the maximum number of retries to open the unix socket.
	MaxRetry uint `hcl:"max_retry,optional"`

	w net.Conn
}

func (p *Unix) NewConnect(context.Context) (io.Writer, error) {
	var err error
	p.w, err = net.Dial("unix", p.Path)
	if err != nil {
		return nil, plugin.PluginError(p, "failed to connect unix socket, path: %s: err: %w", p.Path, err)
	}
	return p.w, nil
}

func (f *Unix) Close(context.Context) {
	f.w.Close()
}

func (p *Unix) MaxConcurrent() uint {
	return math.MaxUint32
}
