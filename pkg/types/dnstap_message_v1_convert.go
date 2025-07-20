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
	"net"
	"strings"
	"time"

	json "github.com/goccy/go-json"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

type dnstapV1Flat struct {
	Timestamp             string `json:"timestamp" msg:"timestamp"`
	QueryTime             string `json:"query_time,omitempty" msg:"query_time"`
	QueryAddress          net.IP `json:"query_address,omitempty" msg:"query_address"`
	QueryAddressHash      string `json:"query_address_hash,omitempty" msg:"query_address_hash"`
	QueryPort             uint32 `json:"query_port,omitempty" msg:"query_port"`
	ResponseTime          string `json:"response_time,omitempty" msg:"response_time"`
	ResponseAddress       net.IP `json:"response_address,omitempty" msg:"response_address"`
	ResponseAddressHash   string `json:"response_address_hash,omitempty" msg:"response_address_hash"`
	ResponsePort          uint32 `json:"response_port,omitempty" msg:"response_port"`
	ResponseZone          string `json:"response_zone,omitempty" msg:"response_zone"`
	EcsNet                string `json:"ecs_net,omitempty" msg:"ecs_net"`
	Identity              string `json:"identity,omitempty" msg:"identity"`
	Type                  string `json:"type" msg:"type"`
	SocketFamily          string `json:"socket_family" msg:"socket_family"`
	SocketProtocol        string `json:"socket_protocol" msg:"socket_protocol"`
	Version               string `json:"version" msg:"version"`
	Extra                 string `json:"extra" msg:"extra"`
	TopLevelDomainName    string `json:"tld" msg:"tld"`
	SecondLevelDomainName string `json:"sld" msg:"sld"`
	ThirdLevelDomainName  string `json:"thirdld" msg:"thirdld"`
	FourthLevelDomainName string `json:"fourthld" msg:"fourthld"`
	Qname                 string `json:"qname" msg:"qname"`
	Qclass                string `json:"qclass" msg:"qclass"`
	Qtype                 string `json:"qtype" msg:"qtype"`
	MessageSize           int    `json:"message_size" msg:"message_size"`
	Txid                  uint16 `json:"txid" msg:"txid"`
	Rcode                 string `json:"rcode" msg:"rcode"`
	AA                    bool   `json:"aa" msg:"aa"`
	TC                    bool   `json:"tc" msg:"tc"`
	RD                    bool   `json:"rd" msg:"rd"`
	RA                    bool   `json:"ra" msg:"ra"`
	AD                    bool   `json:"ad" msg:"ad"`
	CD                    bool   `json:"cd" msg:"cd"`
}

func (d *DnstapMessage) ConvertV1Flat() (*dnstapV1Flat, error) {
	data := &dnstapV1Flat{}

	dt := d.GetDnstap()
	if dt == nil {
		return nil, errors.New("failed to get DNSTAP")
	}
	dnsMsg := d.GetMessage()
	if dnsMsg == nil {
		return nil, errors.New("failed to get DNS message")
	}
	msg := dt.GetMessage()

	data.QueryTime = time.Unix(int64(msg.GetQueryTimeSec()), int64(msg.GetQueryTimeNsec())).Format(time.RFC3339Nano)
	data.QueryAddress = net.IP(msg.GetQueryAddress())
	if v, ok := d.Labels["QueryAddressHash"]; ok {
		data.QueryAddressHash = v
	}
	data.QueryPort = msg.GetQueryPort()

	data.ResponseTime = time.Unix(int64(msg.GetResponseTimeSec()), int64(msg.GetResponseTimeNsec())).Format(time.RFC3339Nano)
	data.ResponseAddress = net.IP(msg.GetResponseAddress())
	if v, ok := d.Labels["ResponseAddressHash"]; ok {
		data.ResponseAddressHash = v
	}
	data.ResponsePort = msg.GetResponsePort()
	data.ResponseZone = string(msg.GetQueryZone())

	if v, ok := d.Labels["ECSQuery"]; ok {
		data.EcsNet = v
	}

	data.Identity = string(dt.GetIdentity())
	data.Type = msg.GetType().String()
	data.SocketFamily = msg.GetSocketFamily().String()
	data.SocketProtocol = msg.GetSocketProtocol().String()
	data.Version = string(dt.GetVersion())

	if len(dnsMsg.Question) > 0 {
		data.Qname = dnsMsg.Question[0].Name
		data.Qclass = dns.ClassToString[dnsMsg.Question[0].Qclass]
		data.Qtype = dns.TypeToString[dnsMsg.Question[0].Qtype]
		labels := strings.Split(dnsMsg.Question[0].Name, ".")

		data.TopLevelDomainName = getName(labels, 2)
		data.SecondLevelDomainName = getName(labels, 3)
		data.ThirdLevelDomainName = getName(labels, 4)
		data.FourthLevelDomainName = getName(labels, 5)
		data.Txid = dnsMsg.MsgHdr.Id

		if msg.ResponseMessage != nil {
			data.MessageSize = len(msg.ResponseMessage)
		} else if msg.QueryMessage != nil {
			data.MessageSize = len(msg.QueryMessage)
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

	return data, nil
}

func (d *DnstapMessage) ConvertV1MapString() (map[string]interface{}, error) {
	res := make(map[string]interface{})

	flat, err := d.ConvertV1Flat()
	if err != nil {
		return nil, err
	}
	res["timestamp"] = flat.Timestamp
	res["query_time"] = flat.QueryTime
	if flat.QueryAddress != nil {
		res["query_address"] = flat.QueryAddress.String()
	}
	res["query_address_hash"] = flat.QueryAddressHash
	res["query_port"] = int64(flat.QueryPort)
	res["response_time"] = flat.ResponseTime
	if flat.ResponseAddress != nil {
		res["response_address"] = flat.ResponseAddress.String()
	}
	res["response_address_hash"] = flat.ResponseAddressHash

	res["response_port"] = int64(flat.ResponsePort)
	res["response_zone"] = flat.ResponseZone
	res["ecs_net"] = flat.EcsNet

	res["identity"] = flat.Identity
	res["type"] = flat.Type
	res["socket_family"] = flat.SocketFamily
	res["socket_protocol"] = flat.SocketProtocol

	res["version"] = flat.Version
	res["extra"] = flat.Extra
	res["tld"] = flat.TopLevelDomainName
	res["sld"] = flat.SecondLevelDomainName
	res["thirdld"] = flat.ThirdLevelDomainName
	res["fourthld"] = flat.FourthLevelDomainName

	res["qname"] = flat.Qname
	res["qclass"] = flat.Qclass
	res["qtype"] = flat.Qtype

	res["message_size"] = int64(flat.MessageSize)
	res["txid"] = int32(flat.Txid)
	res["rcode"] = flat.Rcode

	res["aa"] = flat.AA
	res["tc"] = flat.TC
	res["rd"] = flat.RD
	res["ra"] = flat.RA
	res["ad"] = flat.AD
	res["cd"] = flat.CD

	return res, nil
}

func (d *DnstapMessage) ConvertV1JSON() ([]byte, error) {
	flat, err := d.ConvertV1Flat()
	if err != nil {
		return nil, err
	}
	return json.Marshal(flat)
}

func (d *DnstapMessage) ConvertV1MapStringWithFilter(kf OutputFilters) (map[string]interface{}, error) {
	mapString, err := d.ConvertV1MapString()
	if err != nil {
		return nil, err
	}
	return kf.Filter(mapString), nil
}

func (d *DnstapMessage) ConvertV1JSONWithFilter(kf OutputFilters) ([]byte, error) {
	res, err := d.ConvertV1MapStringWithFilter(kf)
	if err != nil {
		return nil, err
	}
	return json.Marshal(res)
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
