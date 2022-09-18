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

package types

import (
	"fmt"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	"github.com/mimuret/dnsutils/getter"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/proto"
)

type DnstapMessage struct {
	raw    []byte
	dnstap *dnstap.Dnstap
	msg    *dns.Msg

	// for dnstap.Extra
	Labels map[string]string
}

func getDnsMsg(dt *dnstap.Dnstap) (*dns.Msg, error) {
	var dnsRaw []byte
	msg := dt.GetMessage()
	if msg == nil {
		return nil, errors.New("message is invalid")
	}

	if msg.GetQueryMessage() != nil {
		dnsRaw = msg.GetQueryMessage()
	} else {
		dnsRaw = msg.GetResponseMessage()
	}
	if dnsRaw == nil {
		return nil, errors.New("dns msg is an empty")
	}
	dnsMsg := &dns.Msg{}
	if err := dnsMsg.Unpack(dnsRaw); err != nil {
		return nil, errors.New("failed to unpack dns message")
	}
	return dnsMsg, nil
}

func NewDnstapMessage(raw []byte) (*DnstapMessage, error) {
	if raw == nil {
		return nil, errors.New("parameter is nil")
	}
	dt := &dnstap.Dnstap{}
	err := proto.Unmarshal(raw, dt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse DNSTAP message")
	}
	dnsMsg, err := getDnsMsg(dt)
	if err != nil {
		return nil, err
	}
	return &DnstapMessage{raw: raw, dnstap: dt, msg: dnsMsg, Labels: make(map[string]string)}, nil
}

func NewDnstapMessageFromDnstap(dt *dnstap.Dnstap) (*DnstapMessage, error) {
	if dt == nil {
		return nil, errors.New("parameter is nil")
	}
	raw, err := proto.Marshal(dt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to pack DNSTAP message")
	}
	dnsMsg, err := getDnsMsg(dt)
	if err != nil {
		return nil, err
	}
	return &DnstapMessage{raw: raw, dnstap: dt, msg: dnsMsg, Labels: make(map[string]string)}, nil
}

func (d *DnstapMessage) UpdateFromDnstap(dt *dnstap.Dnstap) error {
	if dt == nil {
		return fmt.Errorf("dnstap is nil")
	}
	raw, err := proto.Marshal(dt)
	if err != nil {
		return errors.Wrap(err, "failed to Marshal dnstap")
	}
	dnsMsg, err := getDnsMsg(dt)
	if err != nil {
		return errors.Wrap(err, "failed to Unpack dns msg")
	}

	d.raw = raw
	d.dnstap = dt
	d.msg = dnsMsg
	return nil
}

func (d *DnstapMessage) DeepCopy() *DnstapMessage {
	raw := d.GetRaw()
	cp := &DnstapMessage{
		raw:    make([]byte, len(raw)),
		Labels: maps.Clone(d.Labels),
	}
	copy(cp.raw, d.raw)
	if d.dnstap != nil {
		cp.dnstap = proto.Clone(d.dnstap).(*dnstap.Dnstap)
	}
	if d.msg != nil {
		cp.msg = d.msg.Copy()
	}
	return cp
}

func (d *DnstapMessage) GetDnstap() *dnstap.Dnstap {
	return d.dnstap
}

func (d *DnstapMessage) GetMessage() *dns.Msg {
	return d.msg
}

func (d *DnstapMessage) GetRaw() []byte {
	return d.raw
}

type DnstapMessageGetFunc func(*DnstapMessage) string

func NewGetFuncFromDnstap(f getter.DnstapStrFunc) DnstapMessageGetFunc {
	return func(dm *DnstapMessage) string {
		t := dm.GetDnstap()
		if t == nil {
			return getter.MatchStringUnknown
		}
		return f(t)
	}
}
func NewGetFuncFromDnsMsg(f getter.DnsMsgStrFunc) DnstapMessageGetFunc {
	return func(dm *DnstapMessage) string {
		t := dm.GetMessage()
		if t == nil {
			return getter.MatchStringUnknown
		}
		return f(t)
	}
}
