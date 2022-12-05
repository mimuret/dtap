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
	"sync"

	framestream "github.com/farsightsec/golang-framestream"
	json "github.com/goccy/go-json"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/mimuret/dtap/v2/pkg/logger"
	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/types"

	"github.com/mimuret/dtap/v2/pkg/plugin/input"
	"github.com/mimuret/dtap/v2/pkg/plugin/output"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/nats-io/nats.go"
)

func init() {
	_ = registry.RegisterInputPlugin("nats", Setup)
}

func Setup(bs json.RawMessage) (types.InputPlugin, error) {
	s := &Nats{
		Format:   input.FormatDtapFrame,
		QueueLen: 64,
	}
	if err := json.Unmarshal(bs, s); err != nil {
		return nil, errors.Wrap(err, "failed to decode config")
	}
	if len(s.Hosts) == 0 {
		return nil, errors.Errorf("missing parameter Hosts")
	}
	if s.Subject == "" {
		return nil, errors.Errorf("missing parameter Subject")
	}
	if s.Token == "" && s.User != "" && s.Password == "" {
		return nil, errors.Errorf("missing parameter Password")
	}
	if input.NewInputServer(s.Format, &framestream.DecoderOptions{
		Bidirectional: false,
	}) == nil {
		return nil, errors.Errorf("invalid format")
	}
	return s, nil
}

var _ types.InputPlugin = &Nats{}

type Nats struct {
	plugin.PluginCommon
	sync.Mutex

	*output.DnstapOutput

	// config
	Hosts     []string
	Subject   string
	QueueName string
	QueueLen  int

	User     string
	Password string
	Token    string

	Format input.Format
}

func (f *Nats) Start(ctx context.Context, w types.Writer) error {
LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		default:
			if err := f.Subscribe(ctx, w); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *Nats) Open() (*nats.Conn, error) {

	cfg := nats.GetDefaultOptions()
	cfg.Servers = f.Hosts
	if f.Token != "" {
		cfg.Token = f.Token
	} else if f.User != "" {
		cfg.User = f.User
		cfg.Password = f.Password
	}
	conn, err := cfg.Connect()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create nats producer")
	}
	return conn, nil
}

func (f *Nats) Subscribe(ctx context.Context, w types.Writer) error {
	is := input.NewInputServer(f.Format, &framestream.DecoderOptions{
		Bidirectional: false,
	})
	nc, err := f.Open()
	if err != nil {
		return errors.Wrapf(err, "failed to connect nats server")
	}
	defer func() {
		_ = nc.Drain()
	}()
	ch := make(chan *nats.Msg, f.QueueLen)
	sub, err := nc.ChanQueueSubscribe(f.Subject, f.QueueName, ch)
	if err != nil {
		return errors.Wrapf(err, "failed to subscribe")
	}
	defer func() {
		_ = sub.Unsubscribe()
	}()

	logger.GetLogger().Info("start subscribe", zap.String("subject", f.Subject), zap.String("queue name", f.QueueName), zap.Int("queue len", f.QueueLen))
LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case msg := <-ch:
			buf := bytes.NewBuffer(msg.Data)
			if err := is.Read(buf, w); err != nil {
				return err
			}
		}
	}
	return nil
}
