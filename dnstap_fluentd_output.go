/*
 * Copyright (c) 2018 Manabu Sonoda
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

	"github.com/pkg/errors"

	"github.com/farsightsec/golang-framestream"
	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/golang/protobuf/proto"
)

type DnstapFluentdOutput struct {
	config      *OutputFluentConfig
	fluetConfig fluent.Config
	enc         *framestream.Encoder
	client      *fluent.Fluent
	ipv4Mask    net.IPMask
	ipv6Mask    net.IPMask
	tag         string
}

func NewDnstapFluentdOutput(config *OutputFluentConfig) *DnstapOutput {
	o := &DnstapFluentdOutput{
		config:   config,
		ipv4Mask: net.CIDRMask(config.GetIPv4Mask(), 32),
		ipv6Mask: net.CIDRMask(config.GetIPv6Mask(), 128),
		fluetConfig: fluent.Config{
			FluentHost: config.GetHost(),
			FluentPort: config.GetPort(),
			Async:      false},
		tag: config.GetTag(),
	}
	return NewDnstapOutput(config.GetBufferSize(), o)
}

func (o *DnstapFluentdOutput) open() error {
	var err error
	o.client, err = fluent.New(o.fluetConfig)
	if err != nil {
		return errors.Wrapf(err, "can't create fluent logger")
	}

	return nil
}

func (o *DnstapFluentdOutput) write(frame []byte) error {
	dt := &Dnstap{}
	if err := proto.Unmarshal(frame, dt); err != nil {
		return err
	}
	if data, err := dt.Flat(o.ipv4Mask, o.ipv6Mask); err != nil {
		return err
	} else if err := o.client.Post(o.tag, data); err != nil {
		return errors.Wrapf(err, "failed to post fluent message, tag: %s", o.tag)
	}
	return nil
}

func (o *DnstapFluentdOutput) close() {
	o.client.Close()
}
