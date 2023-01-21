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

	"github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/input"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
)

func init() {
	_ = registry.RegisterInputPlugin("unix", SetupUnixSocket)
}

func SetupUnixSocket(bs json.RawMessage) (types.InputPlugin, error) {
	var err error
	p := &UnixSocket{
		Format: input.FormatDNSTAP,
	}

	if err = json.Unmarshal(bs, p); err != nil {
		return nil, errors.Wrapf(err, "failed to decode config")
	}
	if p.Path == "" {
		return nil, errors.New("missing parameter Path")
	}
	if p.User != "" {
		u, err := user.Lookup(p.User)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get owner name %s", p.User)
		}
		uid, err := strconv.Atoi(u.Uid)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get uid")
		}
		gid, err := strconv.Atoi(u.Gid)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get gid")
		}
		p.uid = &uid
		p.gid = &gid
	}
	if input.NewInputServer(p.Format, nil, nil) == nil {
		return nil, errors.Errorf("invalid format")
	}
	return p, nil
}

var _ types.InputPlugin = &UnixSocket{}

type UnixSocket struct {
	plugin.PluginCommon

	Path   string
	User   string
	Format input.Format

	ln  net.Listener
	uid *int
	gid *int
}

func (p *UnixSocket) Listen() error {
	var err error
	p.ln, err = net.Listen("unix", p.Path)
	if err != nil {
		return errors.Wrapf(err, "failed to listen %s", p.Path)
	}
	if p.uid != nil && p.gid != nil {
		if err := os.Chown(p.Path, *p.uid, *p.gid); err != nil {
			p.ln.Close()
			return errors.Wrapf(err, "failed to change owner %s (%d:%d)", p.User, p.uid, p.gid)
		}
	}
	return nil
}

func (p *UnixSocket) Close() error {
	return p.ln.Close()
}

func (p *UnixSocket) Start(ctx context.Context, ic *types.InputContext) error {
	if err := p.Listen(); err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		p.Close()
	}()
	return input.NewInputServer(p.Format, nil, ic).Serve(p.ln, ic.Writer)
}
