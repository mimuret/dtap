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
	"net"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/goccy/go-json"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"

	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
)

func init() {
	_ = registry.RegisterInputPlugin("pcap", Setup)
}

func Setup(bs json.RawMessage) (types.InputPlugin, error) {
	var err error
	p := &PCAP{
		BPF:       "port 53",
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
	p.bpfInstructionFilter, err = pcap.CompileBPFFilter(layers.LinkTypeEthernet, 65535, p.BPF)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create BPF filter")
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

	// Network device
	Device string
	// BPF Filter
	BPF string

	// If ResolverQueryEnabled is true, input it. Default is false.
	ResolverQueryEnabled bool
	// If ResolverResponseEnabled is true, input it. Default is false.
	ResolverResponseEnabled bool
	// If ClientQueryEnabled is true, input it. Default is false.
	ClientQueryEnabled bool
	// If ClientResponseEnabled is true, input it. Default is false.
	ClientResponseEnabled bool

	WorkerNum int64

	bpfInstructionFilter []pcap.BPFInstruction
	device               *net.Interface
}

func (p *PCAP) Start(ctx context.Context, ic *types.InputContext) error {
	handle, err := pcap.OpenLive(p.Device, 65535, true, pcap.BlockForever)
	if err != nil {
		ic.Logger.Error("failed to open device", zap.Error(err))
		return err
	}
	if err := handle.SetBPFInstructionFilter(p.bpfInstructionFilter); err != nil {
		ic.Logger.Error("failed to set filter", zap.Error(err))
		return err
	}
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	sem := semaphore.NewWeighted(p.WorkerNum)
	ic.Logger.Info("start pcap", zap.String("device", p.Device), zap.String("bpf", p.BPF))
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

func (p *PCAP) handlePacket(ic *types.InputContext, packet gopacket.Packet) {
	dm := &dnstap.Message{}
	dt := &dnstap.Dnstap{
		Type:    dnstap.Dnstap_MESSAGE.Enum(),
		Message: dm,
	}
	l := packet.LinkLayer()
	if l == nil {
		ic.Logger.Debug("failed to get link")
		return
	}
	eth, ok := l.(*layers.Ethernet)
	if !ok {
		ic.Logger.Debug("failed to get layers.Ethernet")
		return
	}
	var send bool
	if bytes.Equal([]byte(eth.SrcMAC), []byte(p.device.HardwareAddr)) {
		send = true
	}

	n := packet.NetworkLayer()
	if n == nil {
		ic.Logger.Debug("failed to get NetworkLayer")
		return
	}
	if ipv4, ok := n.(*layers.IPv4); ok {
		dm.SocketFamily = dnstap.SocketFamily_INET.Enum()
		dm.QueryAddress = ipv4.SrcIP
		dm.ResponseAddress = ipv4.DstIP
	} else if ipv6, ok := n.(*layers.IPv6); ok {
		dm.SocketFamily = dnstap.SocketFamily_INET6.Enum()
		dm.QueryAddress = ipv6.SrcIP
		dm.ResponseAddress = ipv6.DstIP
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
	if udp, ok := t.(*layers.UDP); ok {
		dm.SocketProtocol = dnstap.SocketProtocol_UDP.Enum()
		payload = udp.Payload
		srcPort = uint32(udp.SrcPort)
		dstPort = uint32(udp.DstPort)
	} else if tcp, ok := t.(*layers.TCP); ok {
		dm.SocketProtocol = dnstap.SocketProtocol_TCP.Enum()
		srcPort = uint32(tcp.SrcPort)
		dstPort = uint32(tcp.DstPort)
		if len(tcp.Payload) == 0 {
			return
		}
		payload = tcp.Payload[2:]
	} else {
		ic.Logger.Debug("unknown TransportLayer")
		return
	}
	dm.QueryPort = &srcPort
	dm.ResponsePort = &dstPort
	if send {
		if srcPort == uint32(53) {
			dm.Type = dnstap.Message_CLIENT_RESPONSE.Enum()
			dm.ResponseMessage = payload
		} else {
			dm.Type = dnstap.Message_RESOLVER_QUERY.Enum()
			dm.QueryMessage = payload
		}
	} else {
		if dstPort == uint32(53) {
			dm.Type = dnstap.Message_CLIENT_QUERY.Enum()
			dm.ResponseMessage = payload
		} else {
			dm.Type = dnstap.Message_RESOLVER_RESPONSE.Enum()
			dm.QueryMessage = payload
		}
	}

	switch dm.GetType() {
	case dnstap.Message_CLIENT_QUERY:
		if !p.ClientQueryEnabled {
			return
		}
	case dnstap.Message_CLIENT_RESPONSE:
		if !p.ClientResponseEnabled {
			return
		}
	case dnstap.Message_RESOLVER_QUERY:
		if !p.ResolverQueryEnabled {
			return
		}
	case dnstap.Message_RESOLVER_RESPONSE:
		if !p.ResolverResponseEnabled {
			return
		}
	default:
		return
	}

	frame, err := types.NewDnstapMessageFromDnstap(dt)
	if err != nil {
		ic.Logger.Debug("failed to create DtapFrame", zap.Error(err))
		return
	}
	ic.Writer.Write(frame)
}
