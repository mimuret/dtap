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
package pcap

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"time"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/goccy/go-json"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	gopcapfilter "github.com/packetcap/go-pcap/filter"
	"go.uber.org/zap"
	"golang.org/x/net/bpf"
	"golang.org/x/sync/semaphore"

	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
)

const DNSPort uint32 = 53

func init() {
	_ = registry.RegisterInputPlugin("pcap", Setup)
}

func Setup(bs json.RawMessage) (types.InputPlugin, error) {
	var err error
	p := &PCAP{
		BPF:       "port 53",
		Direction: "inout",
		WorkerNum: 1,
	}
	if err = json.Unmarshal(bs, p); err != nil {
		return nil, errors.Wrapf(err, "failed to decode config")
	}
	if p.Device == "" {
		return nil, errors.New("missing parameter Device")
	}
	if p.device, err = net.InterfaceByName(p.Device); err != nil {
		return nil, errors.Wrapf(err, "missing device %s", p.Device)
	}
	if p.device.HardwareAddr == nil {
		return nil, fmt.Errorf("device does not have a hardware address: %s", p.Device)
	}
	var bpfHw string
	switch p.Direction {
	case "in":
		bpfHw = fmt.Sprintf("ether src host %s", p.device.HardwareAddr)
	case "out":
		bpfHw = fmt.Sprintf("ether dst host %s", p.device.HardwareAddr)
	case "inout":
		bpfHw = fmt.Sprintf("ether host %s", p.device.HardwareAddr)
	default:
		return nil, errors.New("invalid parameter Direction")
	}

	p.bpfFilterStr = fmt.Sprintf("(%s) and (%s)", bpfHw, p.BPF)
	bpfInstructionFilters, err := gopcapfilter.NewExpression(p.bpfFilterStr).Compile().Compile()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create BPF filter")
	}
	for _, v := range bpfInstructionFilters {
		filter, err := v.Assemble()
		if err != nil {
			return nil, errors.Wrap(err, "failed to create BPF filter")
		}
		p.bpfInstructionFilters = append(p.bpfInstructionFilters, filter)
	}
	if p.WorkerNum <= 0 {
		return nil, errors.New("WorkerNum must greater than zero")
	}
	return p, nil
}

var _ types.InputPlugin = &PCAP{}

// The UnixSocket plugin get messages from the unix socket.
type PCAP struct {
	plugin.PluginCommon

	// Device is the network interface to capture packets from.
	Device string

	// BPF is the Berkeley Packet Filter expression to filter packets.
	BPF string

	// Direction specifies the packet direction to capture: `in`, `out`, or `inout`.
	Direction string

	// Enable or disable processing of resolver queries.
	ResolverQueryEnabled bool

	// Enable or disable processing of resolver responses.
	ResolverResponseEnabled bool

	// Enable or disable processing of client queries.
	ClientQueryEnabled bool

	// Enable or disable processing of client responses.
	ClientResponseEnabled bool

	// WorkerNum specifies the number of workers to process packets concurrently.
	// The default value is 1.
	WorkerNum int64

	bpfInstructionFilters []bpf.RawInstruction
	device                *net.Interface
	bpfFilterStr          string
}

func (p *PCAP) Start(ctx context.Context, ic *types.InputContext) error {
	handle, err := pcapgo.NewEthernetHandle(p.device.Name)
	if err != nil {
		ic.Logger.Error("failed to open device", zap.Error(err))
		return err
	}
	if err := handle.SetBPF(p.bpfInstructionFilters); err != nil {
		ic.Logger.Error("failed to set filter", zap.Error(err))
		return err
	}
	packetSource := gopacket.NewPacketSource(handle, layers.LinkTypeEthernet)
	sem := semaphore.NewWeighted(p.WorkerNum)
	ic.Logger.Info("start pcap", zap.String("device", p.Device), zap.String("bpf", p.bpfFilterStr))
LOOP:
	for {
		select {
		case packet := <-packetSource.Packets():
			if err := sem.Acquire(ctx, 1); err != nil {
				ic.Logger.Debug("failed to acquire token")
				continue
			}

			go func(packet gopacket.Packet) {
				p.handlePacket(ic, packet)
				sem.Release(1)
			}(packet)
		case <-ctx.Done():
			break LOOP
		}
	}
	return nil
}

func (p *PCAP) extractLinkLayer(packet gopacket.Packet, ic *types.InputContext) (*layers.Ethernet, bool) {
	l := packet.LinkLayer()
	if l == nil {
		ic.Logger.Debug("failed to get link")
		return nil, false
	}
	eth, ok := l.(*layers.Ethernet)
	if !ok {
		ic.Logger.Debug("failed to get layers.Ethernet")
		return nil, false
	}
	return eth, true
}

// handlePacket processes a single network packet captured by the PCAP plugin.
// It extracts the link layer, network layer, and transport layer information,
// determines the type of DNS message (query or response), and constructs a
// dnstap.Message object. If the message matches the processing criteria, it
// is serialized and written to the output writer.
//
// Parameters:
// - ic: The InputContext containing logger and writer for output.
// - packet: The gopacket.Packet object representing the captured network packet.
//
// The function performs the following steps:
// 1. Extracts the Ethernet layer and checks if the packet is sent by the configured device.
// 2. Extracts the IP layer (IPv4 or IPv6) and determines source and destination addresses.
// 3. Extracts the transport layer (UDP or TCP) and retrieves payload and port information.
// 4. Determines the type of DNS message (client query, client response, resolver query, or resolver response).
// 5. Constructs a dnstap.Message object with the extracted information.
// 6. Writes the message to the output writer if it matches the processing criteria.

func (p *PCAP) handlePacket(ic *types.InputContext, packet gopacket.Packet) {
	dm := &dnstap.Message{}
	dt := &dnstap.Dnstap{
		Type:    dnstap.Dnstap_MESSAGE.Enum(),
		Message: dm,
	}
	eth, ok := p.extractLinkLayer(packet, ic)
	if !ok {
		ic.Logger.Debug("failed to get layers.Ethernet")
		return
	}
	if eth.SrcMAC == nil {
		ic.Logger.Error("failed to get ethernet src mac,can't process packet", zap.String("device", p.Device))
		return
	}
	send := bytes.Equal(eth.SrcMAC, p.device.HardwareAddr)

	n := packet.NetworkLayer()
	if n == nil {
		ic.Logger.Debug("failed to get NetworkLayer")
		return
	}
	var src, dst net.IP
	if ipv4, ok := n.(*layers.IPv4); ok {
		dm.SocketFamily = dnstap.SocketFamily_INET.Enum()
		src = ipv4.SrcIP
		dst = ipv4.DstIP
	} else if ipv6, ok := n.(*layers.IPv6); ok {
		dm.SocketFamily = dnstap.SocketFamily_INET6.Enum()
		src = ipv6.SrcIP
		dst = ipv6.DstIP
	} else {
		ic.Logger.Debug("unknown NetworkLayer")
		return
	}

	t := packet.TransportLayer()
	if t == nil {
		ic.Logger.Debug("failed to get TransportLayer")
		return
	}
	var payload []byte
	var dstPort, srcPort uint32
	switch t := t.(type) {
	case *layers.UDP:
		dm.SocketProtocol = dnstap.SocketProtocol_UDP.Enum()
		payload = t.Payload
		srcPort = uint32(t.SrcPort)
		dstPort = uint32(t.DstPort)
	case *layers.TCP:
		dm.SocketProtocol = dnstap.SocketProtocol_TCP.Enum()
		srcPort = uint32(t.SrcPort)
		dstPort = uint32(t.DstPort)
		if len(t.Payload) < 2 {
			// TCP payload is too short to contain DNS message
			return
		}
		payload = t.Payload[2:]
	default:
		ic.Logger.Debug("unknown TransportLayer")
		return
	}
	timeNow := time.Now()
	timeSec := uint64(timeNow.Unix())
	timeNsec := uint32(timeNow.Nanosecond())
	if send {
		if srcPort == uint32(DNSPort) {
			// resolver:53 -> client:***
			if !p.ClientResponseEnabled {
				return
			}
			dm.Type = dnstap.Message_CLIENT_RESPONSE.Enum()
			dm.ResponseMessage = payload
			dm.QueryAddress = dst
			dm.QueryPort = &dstPort
			dm.ResponseAddress = src
			dm.ResponsePort = &srcPort
			dm.ResponseTimeSec = &timeSec
			dm.ResponseTimeNsec = &timeNsec
		} else {
			// resolver:*** -> auth:***
			if !p.ResolverQueryEnabled {
				return
			}
			dm.Type = dnstap.Message_RESOLVER_QUERY.Enum()
			dm.QueryMessage = payload
			dm.QueryAddress = src
			dm.QueryPort = &srcPort
			dm.ResponseAddress = dst
			dm.ResponsePort = &dstPort
			dm.QueryTimeSec = &timeSec
			dm.QueryTimeNsec = &timeNsec
		}
	} else {
		if dstPort == uint32(DNSPort) {
			// client:*** -> resolver:53
			if !p.ClientQueryEnabled {
				return
			}
			dm.Type = dnstap.Message_CLIENT_QUERY.Enum()
			dm.ResponseMessage = payload
			dm.QueryAddress = src
			dm.QueryPort = &srcPort
			dm.ResponseAddress = dst
			dm.ResponsePort = &dstPort
			dm.QueryTimeSec = &timeSec
			dm.QueryTimeNsec = &timeNsec
		} else {
			// auth:53 -> resolver:***
			if !p.ResolverResponseEnabled {
				return
			}
			dm.Type = dnstap.Message_RESOLVER_RESPONSE.Enum()
			dm.QueryMessage = payload
			dm.QueryAddress = dst
			dm.QueryPort = &dstPort
			dm.ResponseAddress = src
			dm.ResponsePort = &srcPort
			dm.ResponseTimeSec = &timeSec
			dm.ResponseTimeNsec = &timeNsec
		}
	}

	frame, err := types.NewDnstapMessageFromDnstap(dt)
	if err != nil {
		ic.Logger.Debug("failed to create DtapFrame", zap.Error(err))
		return
	}
	ic.Writer.Write(frame)
}
