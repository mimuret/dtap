package multi

import (
	"context"

	"github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
)

func init() {
	_ = registry.RegisterOutputPlugin("multi", Setup)
}

func Setup(bs json.RawMessage) (types.OutputPlugin, error) {
	s := &MultiRunner{
		Concurency: 1,
	}
	if err := json.Unmarshal(bs, s); err != nil {
		return nil, errors.Wrap(err, "failed to decode config")
	}
	if s.Concurency == 0 {
		return nil, errors.New("Concurency must be greater than 0")
	}
	pluginObj := &plugin.PluginCommon{}
	if err := json.Unmarshal(s.Plugin, pluginObj); err != nil {
		return nil, errors.Wrap(err, "failed to decode plugin common config")
	}
	for i := 0; i < int(s.Concurency); i++ {
		p, err := registry.CreateOutputPlugin(pluginObj.GetName(), s.Plugin)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create output plugin %s", s.PluginCommon.Name)
		}
		s.plugns = append(s.plugns, p)
	}
	return s, nil
}

type MultiRunner struct {
	plugin.PluginCommon

	Concurency uint
	Plugin     json.RawMessage

	plugns []types.OutputPlugin
}

var _ types.OutputPlugin = &MultiRunner{}

func (m *MultiRunner) GetName() string {
	return m.PluginCommon.GetName()
}
func (m *MultiRunner) GetID() string {
	return m.PluginCommon.GetID()
}

func (m *MultiRunner) Start(ctx context.Context, oc *types.OutputContext) error {
	for _, p := range m.plugns {
		if err := p.Start(ctx, oc); err != nil {
			return errors.Wrapf(err, "failed to start output plugin %s", p.GetName())
		}
	}
	return nil
}
