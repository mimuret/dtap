/*
 * Copyright (c) 2019 Manabu Sonoda
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

package dtap

import (
	"encoding/json"
	"net"
	"sync"
	"time"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"
	"github.com/golang/protobuf/proto"
	nats "github.com/nats-io/go-nats"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
)

type DnstapNatsOutput struct {
	config     *OutputNatsConfig
	enc        *framestream.Encoder
	con        *nats.Conn
	mux        *sync.Mutex
	dataString []byte
	data       []*DnstapFlatT
	flatOption DnstapFlatOption
	closeCh    chan struct{}
}

func NewDnstapNatsOutput(config *OutputNatsConfig) *DnstapOutput {
	o := &DnstapNatsOutput{
		config: config,
		flatOption: DnstapFlatOption{
			Ipv4Mask:   net.CIDRMask(config.Flat.GetIPv4Mask(), 32),
			Ipv6Mask:   net.CIDRMask(config.Flat.GetIPv6Mask(), 128),
			EnableECS:  config.Flat.GetEnableEcs(),
			IPHashSalt: config.Flat.GetIPHashSalt(),
		},
		data: []*DnstapFlatT{},
		mux:  new(sync.Mutex),
	}
	return NewDnstapOutput(config.Buffer.GetBufferSize(), o)
}

func (o *DnstapNatsOutput) open() error {
	var err error
	if o.config.Token != "" {
		o.con, err = nats.Connect(o.config.GetHost(), nats.Token(o.config.GetToken()))
	} else if o.config.User != "" {
		o.con, err = nats.Connect(o.config.GetHost(), nats.UserInfo(o.config.GetUser(), o.config.GetPassword()))
	} else {
		o.con, err = nats.Connect(o.config.GetHost())
	}
	if err != nil {
		return errors.Wrapf(err, "can't create nats producer")
	}
	o.closeCh = make(chan struct{})
	go o.flush()
	return nil
}

func (o *DnstapNatsOutput) write(frame []byte) error {
	dt := dnstap.Dnstap{}
	if err := proto.Unmarshal(frame, &dt); err != nil {
		return err
	}
	data, err := FlatDnstap(&dt, o.flatOption)
	if err != nil {
		return err
	}
	o.mux.Lock()
	o.data = append(o.data, data)
	o.mux.Unlock()
	return nil
}

func (o *DnstapNatsOutput) flush() {
	ticker := time.NewTicker(10 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			o.publish()
		}
	}
}

func (o *DnstapNatsOutput) publish() {
	o.mux.Lock()
	if len(o.data) == 0 {
		o.mux.Unlock()
		return
	}
	buf, err := json.Marshal(o.data)
	if err != nil {
		log.Debug(err)
		return
	}
	o.data = []*DnstapFlatT{}
	o.mux.Unlock()
	if err := o.con.Publish(o.config.GetSubject(), buf); err != nil {
		log.Warnf("publish error: %v", err)
	}
}

func (o *DnstapNatsOutput) close() {
	close(o.closeCh)
	o.publish()
	o.con.Close()
}
