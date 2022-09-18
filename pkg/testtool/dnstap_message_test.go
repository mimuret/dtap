package testtool_test

import (
	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	"github.com/mimuret/dtap/v2/pkg/testtool"
	"github.com/mimuret/dtap/v2/pkg/types"
)

var (
	dm        *types.DnstapMessage
	tap       *dnstap.Dnstap
	dnsMsgRaw []byte
	msg       *dns.Msg
)

func init() {
	dm = testtool.CreateValidDnstapMessage()
	tap = dm.GetDnstap()
	dnsMsgRaw = tap.GetMessage().GetResponseMessage()
	msg = dm.GetMessage()
	dm.GetRaw()
}
