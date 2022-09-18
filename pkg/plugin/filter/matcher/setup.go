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
package matcher

import (
	"github.com/goccy/go-json"

	umatcher "github.com/mimuret/dnsutils/matcher"
	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
)

func init() {
	_ = registry.RegisterFilterPlugin("matcher", setup)
}

func setup(data json.RawMessage) (types.FilterPlugin, error) {
	g := &Matcher{}
	t := struct {
		plugin.PluginCommon
		Rule umatcher.Config `json:"Rule"`
	}{}
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	set, err := umatcher.BuilderMatchSet(&t.Rule)
	if err != nil {
		return nil, err
	}
	g.PluginCommon = t.PluginCommon
	g.Set = set

	return g, nil
}

var _ types.FilterPlugin = &Matcher{}

type Matcher struct {
	plugin.PluginCommon

	Set *umatcher.MatcherSet `json:"-"`
}

func (f *Matcher) Filter(dt *types.DnstapMessage) *types.DnstapMessage {
	dnstap := dt.GetDnstap()
	dnsmsg := dt.GetMessage()
	if f.Set.Match(dnstap, dnsmsg) {
		return dt
	}
	return nil
}
