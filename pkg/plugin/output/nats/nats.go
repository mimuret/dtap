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
	"fmt"
	"sync"

	json "github.com/goccy/go-json"
	"github.com/mimuret/dtap/v2/pkg/promauto"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

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
	if s.ID == "" {
		return nil, errors.Errorf("`ID` must not be empty")
	}
	s.DnstapOutput = output.NewDnstapOutput(s, s.MaxRetry)

	fmt.Println(s.GetID())
	s.openErr = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "output_nats",
		Name:        "open_errors_total",
		ConstLabels: prometheus.Labels{"ID": s.GetID()},
	})
	s.publishCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "output_nats",
		Name:        "publishes_total",
		ConstLabels: prometheus.Labels{"ID": s.GetID()},
	})
	s.publishErrCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "output_nats",
		Name:        "publish_failed_total",
		ConstLabels: prometheus.Labels{"ID": s.GetID()},
	})
	s.writeMessageCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "output_nats",
		Name:        "write_messages_total",
		ConstLabels: prometheus.Labels{"ID": s.GetID()},
	})
	s.writeMessageErrCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "output_nats",
		Name:        "write_errors_total",
		ConstLabels: prometheus.Labels{"ID": s.GetID()},
	})
	return s, nil
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
	plugin.PluginCommon
	sync.Mutex

	*output.DnstapOutput

	// Hosts is the URL of the nats servers. Must not be empty.
	Hosts []string
	// Nats subject. Must not be empty.
	Subject string

	// Nats user, If a token is given, it is not used.
	User string
	// Nats password, If a token is given, it is not used.
	Password string
	// Nats token
	Token string

	conn *nats.Conn

	// Max nats message size. Default is 1KByte.
	MaxSize int
	// Interval to flush nats messages.
	IntervalSec uint
	// Nats message format.
	Format    pub.Format
	publisher pub.Publisher

	oc *types.OutputContext

	openErr prometheus.Counter

	publishCounter         prometheus.Counter
	publishErrCounter      prometheus.Counter
	writeMessageCounter    prometheus.Counter
	writeMessageErrCounter prometheus.Counter
}

func (f *Nats) SetOutputContext(oc *types.OutputContext) {
	f.oc = oc
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
		f.openErr.Inc()
		return errors.Wrap(err, "failed to create nats producer")
	}
	f.publisher.Start()
	return nil
}

func (f *Nats) Write(dm *types.DnstapMessage) error {
	f.writeMessageCounter.Inc()
	if err := f.publisher.Write(dm); err != nil {
		f.writeMessageErrCounter.Inc()
	}
	return nil
}

func (f *Nats) Publish(data []byte) error {
	err := f.conn.Publish(f.Subject, data)
	f.publishCounter.Inc()
	if err != nil {
		f.publishErrCounter.Inc()
		return errors.Wrap(err, "publish error")
	}
	return nil
}

func (f *Nats) Close() {
	f.publisher.Close()
	f.conn.Close()
}
