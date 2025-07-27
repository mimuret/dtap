package testtool

import (
	"github.com/mimuret/dtap/v3/pkg/types"
)

var _ types.Forwarder = &TestForwader{}

type TestForwader struct {
	Forwarded int
}

func (f *TestForwader) Forward(msg *types.DnstapMessage) {
	f.Forwarded++
}

func (f *TestForwader) SetupForwardTo([]types.Writer) {

}
