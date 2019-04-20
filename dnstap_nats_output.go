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
	"net"

	framestream "github.com/farsightsec/golang-framestream"
	"github.com/golang/protobuf/proto"
	nats "github.com/nats-io/go-nats"
	"github.com/pkg/errors"
)

type DnstapNatsOutput struct {
	config *OutputNatsConfig
	enc    *framestream.Encoder
	con    *nats.EncodedConn

	ipv4Mask net.IPMask
	ipv6Mask net.IPMask
}

func NewDnstapNatsOutput(config *OutputNatsConfig) *DnstapOutput {
	o := &DnstapNatsOutput{
		config:   config,
		ipv4Mask: net.CIDRMask(config.GetIPv4Mask(), 32),
		ipv6Mask: net.CIDRMask(config.GetIPv6Mask(), 128),
	}
	return NewDnstapOutput(config.GetBufferSize(), o)
}

func (o *DnstapNatsOutput) open() error {
	con, err := nats.Connect(o.config.GetHost())
	if err != nil {
		return errors.Wrapf(err, "can't create nats producer")
	}
	o.con, err = nats.NewEncodedConn(con, nats.JSON_ENCODER)
	if err != nil {
		return errors.Wrapf(err, "can't create nats producer")
	}
	return nil
}

func (o *DnstapNatsOutput) write(frame []byte) error {
	dt := &Dnstap{}
	if err := proto.Unmarshal(frame, dt); err != nil {
		return err
	}
	data, err := dt.Flat(o.ipv4Mask, o.ipv6Mask)
	if err != nil {
		return err
	}
	o.con.Publish(o.config.GetSubject(), data)

	return nil
}

func (o *DnstapNatsOutput) close() {
	o.con.Close()
}
