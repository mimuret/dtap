package static

import (
	"github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
)

func init() {
	_ = registry.RegisterFilterPlugin("static", Setup)
}

func Setup(raw json.RawMessage) (types.FilterPlugin, error) {
	fp := &Static{}
	if err := json.Unmarshal(raw, fp); err != nil {
		return nil, err
	}

	return fp, nil
}

var _ types.FilterPlugin = &Static{}

type Static struct {
	plugin.PluginCommon
	Deny bool `json:"Deny"`
}

func (f *Static) Filter(t *types.DnstapMessage) *types.DnstapMessage {
	if f.Deny {
		return nil
	}
	return t
}
