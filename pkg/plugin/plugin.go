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
	json "github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
)

type PluginCommon struct {
	Name string `json:"Name"`
}

func (p *PluginCommon) GetName() string {
	return p.Name
}

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
		ip, err := registry.CreateInputPlugin(cc.Name, raw)
		if err != nil {
			return errors.Wrapf(err, "failed to create Input plugin[%d]", i)
		}
		res = append(res, ip)
	}
	*c = res
	return nil
}

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
		op, err := registry.CreateOutputPlugin(cc.Name, raw)
		if err != nil {
			return errors.Wrapf(err, "failed to create Output plugin[%d]", i)
		}
		res = append(res, op)
	}
	*c = res
	return nil
}

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
		fp, err := registry.CreateFilterPlugin(cc.Name, raw)
		if err != nil {
			return errors.Wrapf(err, "failed to create Filter plugin[%d]", i)
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
