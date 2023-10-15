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
	// Buffer name
	Name string
	// Buffer size
	Size uint
}

func (c *BufferConfig) GetName() string {
	return c.Name
}

func (c *BufferConfig) GetSize() uint {
	return c.Size
}

// An output group consists of a buffer (queue) for the output group,
// Filter settings specific to the output group, and output plug-in settings.
// When a message passes through the global filter and is written to the
// output group's buffer, the global workers runs a filter for the output group.
// Only messages that pass through it are written to the output group's Buffer.
// Each output plugin runs in its own goroutine, receiving and processing
// messages from the output buffer (queue).
// Since output plug-ins are executed in parallel, a message is not processed
// by all output plug-ins, but by one of them. If you want all plug-ins to process
// a message at the same time, for example, standard output and file output, please separate the output groups.
// The current use case for setting up multiple output plugins is to increase
// throughput by setting up multiple plugins with the same configuration, for example, for forwarding to remote.
type OutputGroupConfig struct {
	// Output group name
	Name string
	// Output group shared queue config
	BufferConfig *BufferConfig
	// Filter settings for groups.
	// After filtering, it is added to the output buffer.
	Filters plugin.FilterPlugins
	// Output plugin settings.
	// A message is processed only by one of the plugins.
	Outputs plugin.OutputPlugins
}

// Configuration file structure.
// The configuration file must be in yaml or json format.
// The file extension must be 'yaml' or 'yml' or 'json'.
type Config struct {
	// Number of global workers that process messages.
	// Global workers input messages from global buffers,
	// filter messages with global filters, and copy messages
	// with filter words to output group buffers.
	// default value is 1
	InputFilterWorkerNum uint
	// Output log level.
	// Select one of 'debug', 'info', 'warn', 'error', or 'fatal'.
	// default value is 'info'
	LogLevel string
	// Listen IP and port to output metrics
	ManageHTTPSServer string
	// Input buffer settings
	InputBufferConfig *BufferConfig
	// Input plugin settings. Must not be empty.
	Inputs plugin.InputPlugins
	// The global filters are the filter that is processed for all messages.
	// It can be empty.
	Filters plugin.FilterPlugins
	// Output group settings. Must not be empty.
	// If multiple output groups are specified, messages that pass through
	// the Global Filter are copied to all output groups.
	OutputGroups []OutputGroupConfig
}

func LoadConfig(fs afero.Fs, cfgFile string) (*Config, error) {
	c := &Config{}
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
		ManageHTTPSServer    string
		InputBufferConfig    *BufferConfig
		Inputs               json.RawMessage
		Filters              json.RawMessage
		OutputGroups         []json.RawMessage
	}{
		ManageHTTPSServer: ":9520",
		LogLevel:          "info",
		InputBufferConfig: &BufferConfig{
			Name: "input",
			Size: DefaultInputBufferSize,
		},
		InputFilterWorkerNum: DefaultInputFilterWorkerNum,
	}

	if err := json.Unmarshal(bs, &cfg); err != nil {
		return errors.Wrap(err, "invalid json Input")
	}
	c.InputFilterWorkerNum = cfg.InputFilterWorkerNum
	c.LogLevel = cfg.LogLevel
	c.ManageHTTPSServer = cfg.ManageHTTPSServer
	c.InputBufferConfig = cfg.InputBufferConfig

	var results error

	if err := json.Unmarshal(cfg.Inputs, &c.Inputs); err != nil {
		results = gerrors.Join(results, errors.Wrap(err, "failed to create input plugins"))
	}
	if cfg.Filters != nil {
		if err := json.Unmarshal(cfg.Filters, &c.Filters); err != nil {
			results = gerrors.Join(results, errors.Wrap(err, "failed to create global filter plugins"))
		}
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
