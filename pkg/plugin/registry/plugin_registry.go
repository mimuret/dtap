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
package registry

import (
	"github.com/pkg/errors"

	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/types"
)

var (
	inputPlugins  = map[string]InputPluginSetupFunc{}
	outputPlugins = map[string]OutputPluginSetupFunc{}
	filterPlugins = map[string]FilterPluginSetupFunc{}
)

type InputPluginSetupFunc func(cfg *config.InputBlock) (types.InputPlugin, error)
type OutputPluginSetupFunc func(cfg *config.OutputBlock) (types.OutputPlugin, error)
type FilterPluginSetupFunc func(cfg *config.FilterBlock) (types.FilterPlugin, error)

func RegisterInputPlugin(name string, setupFunc InputPluginSetupFunc) error {
	if name == "" {
		return errors.New("name is an empty")
	}
	if setupFunc == nil {
		return errors.New("setupFunc is nil")
	}
	inputPlugins[name] = setupFunc
	return nil
}

func RegisterOutputPlugin(name string, setupFunc OutputPluginSetupFunc) error {
	if name == "" {
		return errors.New("name is an empty")
	}
	if setupFunc == nil {
		return errors.New("setupFunc is nil")
	}
	outputPlugins[name] = setupFunc
	return nil
}

func RegisterFilterPlugin(name string, setupFunc FilterPluginSetupFunc) error {
	if name == "" {
		return errors.New("name is an empty")
	}
	if setupFunc == nil {
		return errors.New("setupFunc is nil")
	}
	filterPlugins[name] = setupFunc
	return nil
}

func CreateInputPlugin(cfg *config.InputBlock) (types.InputPlugin, error) {
	f, ok := inputPlugins[cfg.GetType()]
	if !ok {
		return nil, errors.Errorf("unknown plugin name `%s`", cfg.GetFullName())
	}
	return f(cfg)
}

func CreateFilterPlugin(cfg *config.FilterBlock) (types.FilterPlugin, error) {
	f, ok := filterPlugins[cfg.GetType()]
	if !ok {
		return nil, errors.Errorf("unknown plugin name `%s`", cfg.GetFullName())
	}
	return f(cfg)
}

func CreateOutputPlugin(cfg *config.OutputBlock) (types.OutputPlugin, error) {
	f, ok := outputPlugins[cfg.GetType()]
	if !ok {
		return nil, errors.Errorf("unknown plugin name `%s`", cfg.GetFullName())
	}
	return f(cfg)
}
