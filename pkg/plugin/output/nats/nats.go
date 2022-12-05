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
	"sync"

	json "github.com/goccy/go-json"
	"github.com/pkg/errors"

	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/types"

	"github.com/mimuret/dtap/v2/pkg/plugin/output"
	"github.com/mimuret/dtap/v2/pkg/plugin/pub"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/nats-io/nats.go"
)

const DefaultMaxPayloadSize = 1024 * 1024
const DnstapFstrmControlHeaderSize = 42
const DnstapFstrmMsgHeaderSize = 4

func init() {
	_ = registry.RegisterOutputPlugin("nats", Setup)
}

func Setup(bs json.RawMessage) (types.OutputPlugin, error) {
	s := &Nats{
		MaxSize:     DefaultMaxPayloadSize,
		Format:      pub.DefaultFormat,
		IntervalSec: 1,
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
	if s.IntervalSec == 0 {
		s.IntervalSec = 1
	}
	s.publisher = pub.NewPublisher(s.Format, s.MaxSize, s.IntervalSec, s)
	if s.publisher == nil {
		return nil, errors.Errorf("failed to create publisher for format %s", s.Format)
	}
	s.DnstapOutput = output.NewDnstapOutput(s)
	return s, nil
}

var _ output.OutputHandler = &Nats{}
var _ types.OutputPlugin = &Nats{}
var _ pub.PublisherHandler = &Nats{}

type Nats struct {
	plugin.PluginCommon
	sync.Mutex

	*output.DnstapOutput

	// config
	Hosts   []string
	Subject string

	User     string
	Password string
	Token    string

	conn *nats.Conn

	MaxSize     int
	IntervalSec uint
	Format      pub.Format
	publisher   pub.Publisher
}

func (f *Nats) Open() error {
	var err error

	cfg := nats.GetDefaultOptions()
	cfg.Servers = f.Hosts
	if f.Token != "" {
		cfg.Token = f.Token
	} else if f.User != "" {
		cfg.User = f.User
		cfg.Password = f.Password
	}
	f.conn, err = cfg.Connect()
	if err != nil {
		return errors.Wrap(err, "failed to create nats producer")
	}
	f.publisher.Start()
	return nil
}

func (f *Nats) Write(dm *types.DnstapMessage) error {
	return f.publisher.Write(dm)
}

func (f *Nats) Publish(data []byte) error {
	err := f.conn.Publish(f.Subject, data)
	return errors.Wrap(err, "publish error")
}

func (f *Nats) Close() {
	f.publisher.Close()
	f.conn.Close()
}
