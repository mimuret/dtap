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
	"time"

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
		return nil, fmt.Errorf("failed to unpack dns message: %w", err)
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
		return nil, fmt.Errorf("failed to parse DNSTAP message: %w", err)
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
		return nil, fmt.Errorf("failed to pack DNSTAP message: %w", err)
	}
	dnsMsg, err := getDnsMsg(dt)
	if err != nil {
		return nil, err
	}
	return &DnstapMessage{raw: raw, dnstap: dt, msg: dnsMsg, Labels: make(map[string]string)}, nil
}

func NewDnstapMessageFromDtapFrame(f *DtapFrame) (*DnstapMessage, error) {
	if f == nil {
		return nil, errors.New("parameter is nil")
	}
	dt := f.GetDnstap()
	if dt == nil {
		return nil, errors.New("Dnstap is nil")
	}
	dm, err := NewDnstapMessageFromDnstap(dt)
	if err != nil {
		return nil, err
	}
	dm.Labels = f.Labels
	return dm, nil
}

func NewDnstapMessageFromDtapFrameRaw(raw []byte) (*DnstapMessage, error) {
	if raw == nil {
		return nil, errors.New("parameter is nil")
	}
	df := &DtapFrame{}
	if err := proto.Unmarshal(raw, df); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DtapFrame: %w", err)
	}
	return NewDnstapMessageFromDtapFrame(df)
}

func (d *DnstapMessage) UpdateFromDnstap(dt *dnstap.Dnstap) error {
	if dt == nil {
		return fmt.Errorf("dnstap is nil")
	}
	raw, err := proto.Marshal(dt)
	if err != nil {
		return fmt.Errorf("failed to Marshal dnstap: %w", err)
	}
	dnsMsg, err := getDnsMsg(dt)
	if err != nil {
		return fmt.Errorf("failed to Unpack dns msg: %w", err)
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

func (d *DnstapMessage) ToDtapFrame() *DtapFrame {
	return &DtapFrame{
		Dnstap: d.GetDnstap(),
		Labels: d.Labels,
	}
}

func (d *DnstapMessage) GetTimestamp() *time.Time {
	res := d.GetResponseTime()
	if res != nil {
		return res
	}
	return d.GetQueryTime()
}

func (d *DnstapMessage) GetQueryTime() *time.Time {
	msg := d.GetDnstap()
	if d.GetDnstap() == nil {
		return nil
	}
	if msg.Message.QueryTimeSec == nil {
		return nil
	}
	res := time.Unix(int64(msg.Message.GetQueryTimeSec()), int64(msg.Message.GetQueryTimeNsec()))
	return &res
}

func (d *DnstapMessage) GetResponseTime() *time.Time {
	msg := d.GetDnstap()
	if d.GetDnstap() == nil {
		return nil
	}
	if msg.Message.ResponseTimeSec == nil {
		return nil
	}
	res := time.Unix(int64(msg.Message.GetResponseTimeSec()), int64(msg.Message.GetResponseTimeNsec()))
	return &res
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
