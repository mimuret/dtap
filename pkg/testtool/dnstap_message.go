package testtool

import (
	"net"
	"time"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	dnsutilstesttool "github.com/mimuret/dnsutils/testtool"
	"github.com/mimuret/dtap/v2/pkg/types"
	"google.golang.org/protobuf/proto"
)

func CreateValidDnstapMessage() *types.DnstapMessage {
	msg := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:            0,
			Response:      true,
			Opcode:        dns.OpcodeQuery,
			Authoritative: true,
			Rcode:         dns.RcodeSuccess,
		},
		Question: []dns.Question{
			{Name: "example.jp.", Qclass: dns.ClassINET, Qtype: dns.TypeA},
		},
		Answer: []dns.RR{
			dnsutilstesttool.MustNewRR("example.jp. 300 IN A 127.0.0.1"),
		},
	}
	msgRaw, err := msg.Pack()
	if err != nil {
		panic(err)
	}
	port1024 := uint32(1024)
	port443 := uint32(443)
	now := time.Now()
	sec := uint64(now.Unix())
	nsec := uint32(now.UnixNano())
	dt := &dnstap.Dnstap{
		Identity: []byte("localhost."),
		Version:  []byte("dnstap"),
		Type:     dnstap.Dnstap_MESSAGE.Enum(),
		Message: &dnstap.Message{
			Type:             dnstap.Message_AUTH_RESPONSE.Enum(),
			SocketFamily:     dnstap.SocketFamily_INET.Enum(),
			SocketProtocol:   dnstap.SocketProtocol_DOH.Enum(),
			QueryAddress:     net.IPv4(192, 168, 0, 1).To4(),
			ResponseAddress:  net.IPv4(10, 0, 0, 1).To4(),
			QueryPort:        &port443,
			ResponsePort:     &port1024,
			QueryTimeSec:     nil,
			QueryTimeNsec:    nil,
			QueryMessage:     nil,
			QueryZone:        nil,
			ResponseTimeSec:  &sec,
			ResponseTimeNsec: &nsec,
			ResponseMessage:  msgRaw,
		},
	}
	raw, err := proto.Marshal(dt)
	if err != nil {
		panic(err)
	}
	dm, err := types.NewDnstapMessage(raw)
	if err != nil {
		panic(err)
	}
	return dm
}
