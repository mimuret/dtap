package nop

import (
	"github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
)

func init() {
	_ = registry.RegisterFilterPlugin("nop", Setup)
}

func Setup(raw json.RawMessage) (types.FilterPlugin, error) {
	return &Nop{plugin.PluginCommon{Name: "nop"}}, nil
}

var _ types.FilterPlugin = &Nop{}

type Nop struct {
	plugin.PluginCommon
}

func (f *Nop) Filter(t *types.DnstapMessage) *types.DnstapMessage {
	return t
}
