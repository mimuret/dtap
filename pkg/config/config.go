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

package config

import (
	"fmt"
	"path"

	json "github.com/goccy/go-json"
	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/spf13/afero"

	gerrors "errors"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

const DefaultInputBufferSize = 10000
const DefaultOutputBufferSize = 10000
const DefaultInputFilterWorkerNum = 1

type BufferConfig struct {
	Name string
	Size uint
}

func (c *BufferConfig) GetName() string {
	return c.Name
}

func (c *BufferConfig) GetSize() uint {
	return c.Size
}

type OutputGroupConfig struct {
	Name         string
	BufferConfig *BufferConfig
	Filters      plugin.FilterPlugins
	Outputs      plugin.OutputPlugins
}

type Config struct {
	// number of workers that moves msg from input buffer to output buffer with msg filtering.
	InputFilterWorkerNum uint
	LogLevel             string
	MetricsListen        string
	InputBufferConfig    *BufferConfig
	Inputs               plugin.InputPlugins
	Filters              plugin.FilterPlugins
	OutputGroups         []OutputGroupConfig
}

func NewConfig() *Config {
	return &Config{
		MetricsListen: ":9520",
		LogLevel:      "info",
		InputBufferConfig: &BufferConfig{
			Name: "input",
			Size: DefaultInputBufferSize,
		},
		InputFilterWorkerNum: DefaultInputFilterWorkerNum,
	}
}

func LoadConfig(fs afero.Fs, cfgFile string) (*Config, error) {
	c := NewConfig()
	bs, err := afero.ReadFile(fs, cfgFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open config file")
	}
	switch path.Ext(cfgFile) {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(bs, c)
	case ".json":
		err = json.Unmarshal(bs, c)
	default:
		return nil, errors.New("unsupported config format")
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed parse config file")
	}

	OutputGroupName := map[string]struct{}{}
	for i := range c.OutputGroups {
		og := &c.OutputGroups[i]
		if og.Name == "input" {
			return nil, fmt.Errorf("invalid parameter OutputGroups[%d].Name must not input", i)
		}
		if og.Name == "" {
			og.Name = fmt.Sprintf("output-group-%d", i)
		}
		if _, exist := OutputGroupName[og.Name]; exist {
			return nil, fmt.Errorf("invalid parameter OutputGroups[%d].Name `%s` is already exist", i, og.Name)
		}
		if og.BufferConfig == nil {
			og.BufferConfig = &BufferConfig{
				Size: DefaultInputBufferSize,
			}
		}
		og.BufferConfig.Name = og.Name
		OutputGroupName[og.Name] = struct{}{}
	}
	return c, nil
}

func (c *Config) UnmarshalJSON(bs []byte) error {
	cfg := struct {
		InputFilterWorkerNum uint
		LogLevel             string
		MetricsListen        string
		InputBufferConfig    *BufferConfig
		Inputs               json.RawMessage
		Filters              json.RawMessage
		OutputGroups         []json.RawMessage
	}{}

	if err := json.Unmarshal(bs, &cfg); err != nil {
		return errors.Wrap(err, "invalid json Input")
	}
	c.InputFilterWorkerNum = cfg.InputFilterWorkerNum
	c.LogLevel = cfg.LogLevel
	c.MetricsListen = cfg.MetricsListen
	c.InputBufferConfig = cfg.InputBufferConfig

	var results error

	if err := json.Unmarshal(cfg.Inputs, &c.Inputs); err != nil {
		results = gerrors.Join(results, errors.Wrap(err, "failed to create input plugins"))
	}
	if err := json.Unmarshal(cfg.Filters, &c.Filters); err != nil {
		results = gerrors.Join(results, errors.Wrap(err, "failed to create global filter plugins"))
	}
	for i, v := range cfg.OutputGroups {
		var og OutputGroupConfig
		if err := json.Unmarshal(v, &og); err != nil {
			results = gerrors.Join(results, errors.Wrapf(err, "failed to create output groups no %d", i))
		}
		c.OutputGroups = append(c.OutputGroups, og)
	}

	return results
}
