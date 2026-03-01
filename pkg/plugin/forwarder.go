package plugin

import (
	"github.com/mimuret/dtap/v3/pkg/types"
)

type Forwarder struct {
	forwardTo   []types.Writer
	constLabels map[string]string
}

func NewForwarder(constLabels map[string]string) *Forwarder {
	return &Forwarder{
		constLabels: constLabels,
	}
}

func (f *Forwarder) SetupForwardTo(writers []types.Writer) {
	f.forwardTo = writers
}

func (f *Forwarder) Forward(m *types.DnstapMessage) {
	if f.constLabels != nil {
		for k, v := range f.constLabels {
			m.Labels[k] = v
		}
	}
	for _, w := range f.forwardTo {
		w.Write(m)
	}
}
