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
	"crypto/sha256"
	"fmt"
	"net"
	"strings"
	"time"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

type DnstapFlatT struct {
	Timestamp             string `json:"timestamp"`
	QueryTime             string `json:"query_time,omitempty"`
	QueryAddress          net.IP `json:"query_address,omitempty"`
	QueryAddressHash      string `json:"query_address_hash,omitempty"`
	QueryPort             uint32 `json:"query_port,omitempty"`
	ResponseTime          string `json:"response_time,omitempty"`
	ResponseAddress       net.IP `json:"response_address,omitempty"`
	ResponseAddressHash   string `json:"response_address,omitempty"`
	ResponsePort          uint32 `json:"response_port,omitempty"`
	ResponseZone          string `json:"response_zone,omitempty"`
	EcsNet                *Net   `json:"ecs_net,omitempty"`
	Identity              string `json:"identity,omitempty"`
	Type                  string `json:"type"`
	SocketFamily          string `json:"socket_family"`
	SocketProtocol        string `json:"socket_protocol"`
	Version               string `json:"version"`
	Extra                 string `json:"extra"`
	TopLevelDomainName    string `json:"tld"`
	SecondLevelDomainName string `json:"2ld"`
	ThirdLevelDomainName  string `json:"3ld"`
	FourthLevelDomainName string `json:"4ld"`
	Qname                 string `json:"qname"`
	Qclass                string `json:"qclass"`
	Qtype                 string `json:"qtype"`
	MessageSize           int    `json:"message_size"`
	Txid                  uint16 `json:"txid"`
	Rcode                 string `json:"rcode"`
	AA                    bool   `json:"aa"`
	TC                    bool   `json:"tc"`
	RD                    bool   `json:"rd"`
	RA                    bool   `json:"ra"`
	AD                    bool   `json:"ad"`
	CD                    bool   `json:"cd"`
}

var (
	DefaultIPv4Mask = net.CIDRMask(22, 22)
	DefaultIPv6Mask = net.CIDRMask(40, 40)
)

type DnstapFlatOption interface {
	GetIPv4Mask() net.IPMask
	GetIPv6Mask() net.IPMask
	GetEnableEcs() bool
	GetEnableHashIP() bool
	GetIPHashSalt() []byte
}

func FlatDnstap(dt *dnstap.Dnstap, opt DnstapFlatOption) (*DnstapFlatT, error) {
	var data = DnstapFlatT{}
	h := sha256.New()

	var dnsMessage []byte
	msg := dt.GetMessage()
	if msg.GetQueryMessage() != nil {
		dnsMessage = msg.GetQueryMessage()
	} else {
		dnsMessage = msg.GetResponseMessage()
	}

	data.QueryTime = time.Unix(int64(msg.GetQueryTimeSec()), int64(msg.GetQueryTimeNsec())).Format(time.RFC3339Nano)
	data.ResponseTime = time.Unix(int64(msg.GetResponseTimeSec()), int64(msg.GetResponseTimeNsec())).Format(time.RFC3339Nano)
	if len(msg.GetQueryAddress()) == 4 {
		data.QueryAddress = net.IP(msg.GetQueryAddress()).Mask(opt.GetIPv4Mask())
	} else {
		data.QueryAddress = net.IP(msg.GetQueryAddress()).Mask(opt.GetIPv6Mask())
	}
	if opt.GetEnableHashIP() && opt.GetIPHashSalt() != nil {
		bs := make([]byte, len(opt.GetIPHashSalt())+16)
		bs = append(bs, opt.GetIPHashSalt()...)
		bs = append(bs, net.IP(msg.GetQueryAddress()).To16()...)
		data.QueryAddressHash = fmt.Sprintf("%x", h.Sum(bs))
	}
	data.QueryPort = msg.GetQueryPort()
	if len(msg.GetResponseAddress()) == 4 {
		data.ResponseAddress = net.IP(msg.GetResponseAddress()).Mask(opt.GetIPv4Mask()).To4()
	} else {
		data.ResponseAddress = net.IP(msg.GetResponseAddress()).Mask(opt.GetIPv6Mask()).To16()
	}
	if opt.GetEnableHashIP() && opt.GetIPHashSalt() != nil {
		bs := make([]byte, len(opt.GetIPHashSalt())+16)
		bs = append(bs, opt.GetIPHashSalt()...)
		bs = append(bs, net.IP(msg.GetResponseAddress()).To16()...)
		data.ResponseAddressHash = fmt.Sprintf("%x", h.Sum(bs))
	}

	data.ResponsePort = msg.GetResponsePort()
	data.ResponseZone = string(msg.GetQueryZone())
	data.Identity = string(dt.GetIdentity())
	if data.Identity == "" {
		data.Identity = hostname
	}
	data.Type = msg.GetType().String()
	data.SocketFamily = msg.GetSocketFamily().String()
	data.SocketProtocol = msg.GetSocketProtocol().String()
	data.Version = string(dt.GetVersion())
	data.Extra = string(dt.GetExtra())
	dnsMsg := dns.Msg{}
	if err := dnsMsg.Unpack(dnsMessage); err != nil {
		return nil, errors.Wrapf(err, "can't parse dns message() failed: %s\n", err)
	}

	if len(dnsMsg.Question) > 0 {
		data.Qname = dnsMsg.Question[0].Name
		data.Qclass = dns.ClassToString[dnsMsg.Question[0].Qclass]
		data.Qtype = dns.TypeToString[dnsMsg.Question[0].Qtype]
		labels := strings.Split(dnsMsg.Question[0].Name, ".")

		data.TopLevelDomainName = getName(labels, 2)
		data.SecondLevelDomainName = getName(labels, 3)
		data.ThirdLevelDomainName = getName(labels, 4)
		data.FourthLevelDomainName = getName(labels, 5)

		data.MessageSize = len(dnsMessage)
		data.Txid = dnsMsg.MsgHdr.Id
	}
	if opt.GetEnableEcs() {
		if len(dnsMsg.Extra) > 0 {
			for _, rr := range dnsMsg.Extra {
				if optrr, ok := rr.(*dns.OPT); ok {
					for _, edns0opt := range optrr.Option {
						if ecs, ok := edns0opt.(*dns.EDNS0_SUBNET); ok {
							ip := ecs.Address
							// ipv4
							if ecs.Family == 1 {
								ip = ip.Mask(opt.GetIPv4Mask())
							} else {
								ip = ip.Mask(opt.GetIPv6Mask())
							}
							data.EcsNet = &Net{
								IP:           ip,
								PrefixLength: int(ecs.SourceNetmask),
							}
						}
					}
				}
			}
		}
	}
	data.Rcode = dns.RcodeToString[dnsMsg.Rcode]
	data.AA = dnsMsg.Authoritative
	data.TC = dnsMsg.Truncated
	data.RD = dnsMsg.RecursionDesired
	data.RA = dnsMsg.RecursionAvailable
	data.AD = dnsMsg.AuthenticatedData
	data.CD = dnsMsg.CheckingDisabled

	switch msg.GetType() {
	case dnstap.Message_AUTH_QUERY, dnstap.Message_RESOLVER_QUERY,
		dnstap.Message_CLIENT_QUERY, dnstap.Message_FORWARDER_QUERY,
		dnstap.Message_STUB_QUERY, dnstap.Message_TOOL_QUERY:
		data.Timestamp = data.QueryTime
	case dnstap.Message_AUTH_RESPONSE, dnstap.Message_RESOLVER_RESPONSE,
		dnstap.Message_CLIENT_RESPONSE, dnstap.Message_FORWARDER_RESPONSE,
		dnstap.Message_STUB_RESPONSE, dnstap.Message_TOOL_RESPONSE:
		data.Timestamp = data.ResponseTime
	}

	return &data, nil
}

func getName(labels []string, i int) string {
	var res string
	labelsLen := len(labels)
	if labelsLen-i >= 0 {
		res = strings.Join(labels[labelsLen-i:labelsLen-1], ".")
	} else {
		res = strings.Join(labels, ".")
	}
	return res
}
