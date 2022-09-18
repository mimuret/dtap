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

package fluent

import (
	"github.com/fluent/fluent-logger-golang/fluent"
	json "github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/output"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
)

func init() {
	_ = registry.RegisterOutputPlugin("fluent", Setup)
}

func Setup(bs json.RawMessage) (types.OutputPlugin, error) {
	s := &Fluent{}
	if err := json.Unmarshal(bs, s); err != nil {
		return nil, errors.Wrap(err, "failed to decode config")
	}
	s.DnstapOutput = output.NewDnstapOutput(s)
	return s, nil
}

var _ types.OutputPlugin = &Fluent{}

type Fluent struct {
	plugin.PluginCommon
	*output.DnstapOutput

	// Coonfig
	FluentConfig fluent.Config
	Tag          string

	// fluent
	client *fluent.Fluent
}

func (o *Fluent) Open() error {
	var err error
	o.client, err = fluent.New(o.FluentConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create fluent logger: %w")
	}

	return nil
}

func (o *Fluent) Write(dm *types.DnstapMessage) error {
	data, err := dm.ConvertV1Flat()
	if err != nil {
		return err
	}
	if err := o.client.Post(o.Tag, data); err != nil {
		return errors.Wrapf(err, "failed to post fluent message, tag: %s", o.Tag)
	}
	return nil
}

func (o *Fluent) Close() {
	o.client.Close()
}
