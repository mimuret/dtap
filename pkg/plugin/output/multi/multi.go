package multi

import (
	"context"
	"fmt"

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
	var pluginName string
	if err := json.Unmarshal(s.Plugin["Name"], &pluginName); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal Plugin.Name")
	}
	if pluginName == "" {
		return nil, errors.New("`Plugin.Name` must not be empty")
	}
	for i := 0; i < int(s.Concurency); i++ {
		var err error
		s.Plugin["ID"], err = json.Marshal(fmt.Sprintf(`%s-%d`, s.GetID(), i))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal Plugin.ID for concurrency %d", i)
		}
		cfg, err := json.Marshal(s.Plugin)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal plugin config: %s", pluginName)
		}
		p, err := registry.CreateOutputPlugin(pluginName, cfg)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create output plugin %s", s.PluginCommon.Name)
		}
		s.plugns = append(s.plugns, p)
	}
	return s, nil
}

// MultiRunner is a plugin that creates multiple instances of an Output Plugin
// defined in the `Plugin` field. It is useful for load balancing or distributing
// output to multiple destinations.
//
// The number of instances is controlled by the `Concurency` field. Each instance
// of the plugin will have a unique ID, which is derived from the MultiRunner's ID
// followed by a hyphen and a sequential number starting from 0.
//
// For example, the following configurations are equivalent:
//
// Configuration 1:
//   - Name: multi
//		 ID: test-multi
//     Concurency: 2
//     Plugin:
//     Name: dummy
//
// Configuration 2:
//   - Name: dummy
//     ID: test-multi-0
//   - Name: dummy
//     ID: test-multi-1
//
// Fields:
// - `Concurency`: Specifies the number of concurrent plugin instances to create.
// - `Plugin`: Defines the configuration for the child plugin to be instantiated.
//
// The `Start` method initializes all created plugins, and errors are returned if
// any of the plugins fail to start.

type MultiRunner struct {
	plugin.PluginCommon

	Concurency uint
	Plugin     map[string]json.RawMessage

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
