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

	"github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/input"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
)

func init() {
	_ = registry.RegisterInputPlugin("tcp", SetupTCPSocket)
}

func SetupTCPSocket(bs json.RawMessage) (types.InputPlugin, error) {
	p := &TCPSocket{
		Format: input.FormatDNSTAP,
	}

	if err := json.Unmarshal(bs, p); err != nil {
		return nil, errors.Wrap(err, "failed to decode config")
	}
	if p.Port == 0 {
		return nil, errors.Errorf("missing parameter Port")
	}
	if input.NewInputServer(p.Format, nil, nil) == nil {
		return nil, errors.Errorf("invalid format")
	}
	return p, nil
}

var _ types.InputPlugin = &TCPSocket{}

// The TCPSocket plugin get messages from the tcp socket.
type TCPSocket struct {
	plugin.PluginCommon

	// Listen Address. If not given, I will listen to any.
	Address string
	// Listen port. Must not be empty.
	Port uint16
	// Message format
	Format input.Format

	ln net.Listener
}

func (p *TCPSocket) Listen() error {
	var err error
	target := net.JoinHostPort(p.Address, strconv.Itoa(int(p.Port)))
	p.ln, err = net.Listen("tcp", target)
	if err != nil {
		return errors.Wrapf(err, "failed to listen %s", target)
	}
	return nil
}

func (p *TCPSocket) Close() error {
	return p.ln.Close()
}

func (p *TCPSocket) Start(ctx context.Context, ic *types.InputContext) error {
	if err := p.Listen(); err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		p.Close()
	}()
	return input.NewInputServer(p.Format, nil, ic).Serve(p.ln, ic.Writer)
}
