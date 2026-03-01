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
	"net"
	"os"
	"os/user"
	"strconv"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/hashicorp/hcl/v2/gohcl"
	"go.uber.org/zap"

	"errors"

	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/plugin/input"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/mimuret/dtap/v3/pkg/types"
)

const PLUGIN_NAME = "unix"

func init() {
	_ = registry.RegisterInputPlugin(PLUGIN_NAME, Setup)
}

func Setup(cfg *config.InputBlock) (types.InputPlugin, error) {
	p := &UnixSocket{
		InputBlock: *cfg,
	}
	// Decode the HCL body into the UnixSocket struct.
	diags := gohcl.DecodeBody(cfg.Body, nil, p)
	if diags.HasErrors() {
		return nil, plugin.PluginError(p, "failed to setup unix plugin: %w", errors.Join(diags.Errs()...))
	}
	if p.Path == "" {
		return nil, plugin.PluginError(p, "missing required parameter: Path")
	}
	if p.User != "" {
		u, err := user.Lookup(p.User)
		if err != nil {
			return nil, plugin.PluginError(p, "failed to lookup user '%s' err: %w", p.User, err)
		}
		uid, err := strconv.Atoi(u.Uid)
		if err != nil {
			return nil, plugin.PluginError(p, "failed to get uid user: %s err: %w", p.User, err)
		}
		gid, err := strconv.Atoi(u.Gid)
		if err != nil {
			return nil, plugin.PluginError(p, "failed to get gid:user: %s err: %w", p.User, err)
		}
		p.uid = &uid
		p.gid = &gid
	}
	p.is = input.NewInputServer(p, p.Format, nil)
	if p.is == nil {
		return nil, plugin.PluginError(p, "invalid format: %s", p.Format)
	}
	return p, nil
}

var _ types.InputPlugin = &UnixSocket{}

// The UnixSocket plugin get messages from the unix socket.

// Example HCL configuration:
// ```hcl
//
// filter "unix" "default" {
//   path = "/var/run/dnstap.sock"
//   user = "dnstap"
// }
//
// ```

// UnixSocket represents a plugin that listens for DNSTAP messages over a Unix socket.
type UnixSocket struct {
	config.InputBlock

	// Path specifies the Unix socket file path.
	Path string `hcl:"path"`

	// User specifies the owner of the Unix socket file. Optional.
	User string `hcl:"user,optional"`

	// Format specifies the message format. Optional, defaults to "DNSTAP".
	Format string `hcl:"format,optional"`

	// uid and gid store the user ID and group ID for the socket owner.
	uid *int
	gid *int

	// is is the input server instance.
	is *input.InputServer
}

func (p *UnixSocket) Listen() (net.Listener, error) {
	ln, err := net.Listen("unix", p.Path)
	if err != nil {
		return nil, plugin.PluginError(p, "failed to listen %s: err: %w", p.Path, err)
	}
	if p.uid != nil && p.gid != nil {
		if err := os.Chown(p.Path, *p.uid, *p.gid); err != nil {
			ln.Close()
			os.Remove(p.Path) // ソケットファイルを削除
			return nil, plugin.PluginError(p, "failed to change owner %s (%d:%d): %w", p.User, *p.uid, *p.gid, err)
		}
	}
	return ln, nil
}

func (p *UnixSocket) Start(ctx context.Context, forwarder types.Forwarder) error {
	ln, err := p.Listen()
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		if err := ln.Close(); err != nil {
			ctxzap.Error(ctx, "failed to close UnixSocket listener", zap.String("path", p.Path), zap.Error(err))
		}
		os.Remove(p.Path)
	}()
	return p.is.Serve(ctx, forwarder, ln)
}
