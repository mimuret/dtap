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
package plugin

import (
	"fmt"

	json "github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
)

type PluginCommon struct {
	// Plugin type name
	Name string `json:"Name"`
	// Plugin type id
	ID string `json:"ID"`
	// Maximum number of retries. A value of 0 means infinite.
	MaxRetry uint `json:"MaxRetry"`
}

func (p *PluginCommon) GetName() string {
	return p.Name
}

func (p *PluginCommon) GetID() string {
	return p.Name
}

var (
	ErrAlreadyExist = fmt.Errorf("already exist plugin")
	registerPlugins = map[string]types.Plugin{}
)

func registerID(id string, p types.Plugin) error {
	if ap, exist := registerPlugins[id]; exist {
		return fmt.Errorf("plugin with `ID` `%s` is a duplicate of Plugin with `Name` %s", ap.GetID(), ap.GetName())
	}
	registerPlugins[id] = p
	return nil
}

// Input plugin slices
// If multiple Plugins are specified, they are started in order, but input processing is performed in parallel.
type InputPlugins []types.InputPlugin

func (c *InputPlugins) UnmarshalJSON(bs []byte) error {
	res := InputPlugins{}
	raws := []json.RawMessage{}
	if err := json.Unmarshal(bs, &raws); err != nil {
		return errors.Wrap(err, "invalid json Input")
	}
	for i, raw := range raws {
		cc := &PluginCommon{}
		if err := json.Unmarshal(raw, cc); err != nil {
			return errors.Wrapf(err, "invalid json Input[%d]", i)
		}
		if cc.ID == "" {
			return errors.Errorf("Input[%d].ID must not be empty", i)
		}
		ip, err := registry.CreateInputPlugin(cc.Name, raw)
		if err != nil {
			return errors.Wrapf(err, "failed to create input plugin, no %d, name is `%s`", i, cc.Name)
		}
		if err := registerID(cc.ID, ip); err != nil {
			return err
		}
		res = append(res, ip)
	}
	*c = res
	return nil
}

// Output plugin slices
// If multiple plugins are specified, they are started in sequence and output processing is performed in parallel.
// In other words, a message is processed only by one of the plugins.
type OutputPlugins []types.OutputPlugin

func (c *OutputPlugins) UnmarshalJSON(bs []byte) error {
	res := OutputPlugins{}
	raws := []json.RawMessage{}
	if err := json.Unmarshal(bs, &raws); err != nil {
		return errors.Wrap(err, "invalid json Output")
	}

	for i, raw := range raws {
		cc := &PluginCommon{}
		if err := json.Unmarshal(raw, cc); err != nil {
			return errors.Wrapf(err, "invalid json Output[%d]", i)
		}
		if cc.ID == "" {
			return errors.Errorf("Outputs[%d].ID must not be empty", i)
		}
		op, err := registry.CreateOutputPlugin(cc.Name, raw)
		if err != nil {
			return errors.Wrapf(err, "failed to create output plugin, no %d, name is `%s`", i, cc.Name)
		}
		if err := registerID(cc.ID, op); err != nil {
			return err
		}
		res = append(res, op)
	}
	*c = res
	return nil
}

// If multiple plugins are specified, they are filtered in sequence;
// a single message is processed in series, not in parallel.
type FilterPlugins []types.FilterPlugin

func (c *FilterPlugins) UnmarshalJSON(bs []byte) error {
	res := FilterPlugins{}
	raws := []json.RawMessage{}
	if err := json.Unmarshal(bs, &raws); err != nil {
		return errors.Wrap(err, "invalid json Filter")
	}

	for i, raw := range raws {
		cc := &PluginCommon{}
		if err := json.Unmarshal(raw, cc); err != nil {
			return errors.Wrapf(err, "invalid json Filter[%d]", i)
		}
		if cc.ID == "" {
			return errors.Errorf("Filters[%d].ID must not be empty", i)
		}
		fp, err := registry.CreateFilterPlugin(cc.Name, raw)
		if err != nil {
			return errors.Wrapf(err, "failed to create filter plugin, no %d, name is `%s`", i, cc.Name)
		}
		if err := registerID(cc.ID, fp); err != nil {
			return err
		}
		res = append(res, fp)
	}
	*c = res
	return nil
}

func (c FilterPlugins) Filter(dm *types.DnstapMessage) *types.DnstapMessage {
	for _, filterPlugin := range c {
		dm = filterPlugin.Filter(dm)
		if dm == nil {
			return nil
		}
	}
	return dm
}
