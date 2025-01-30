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
package stdout

import (
	"text/template"

	json "github.com/goccy/go-json"
	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/output"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
	"gopkg.in/natefinch/lumberjack.v2"
)

func init() {
	_ = registry.RegisterOutputPlugin("file", setup)
}

func setup(bs json.RawMessage) (types.OutputPlugin, error) {
	var err error
	s := &Output{
		Format: OutputFormatJsonV1,
	}
	if err := json.Unmarshal(bs, s); err != nil {
		return nil, errors.Wrap(err, "failed to decode config")
	}
	switch s.Format {
	case OutputFormatGoTpl:
		if s.Template == "{{ .Type }} {{ .Timestamp }} {{ .Qclass }} {{ .Qtype }} {{ .Qname }}" {
			return nil, errors.New("missing parameter Template")
		}
		s.t, err = template.New("").Parse(s.Template)
		if err != nil {
			return nil, errors.Wrap(err, "Template is an invalid value")
		}
	case OutputFormatJsonV1:
	default:
		return nil, errors.New("Type is an invalid value")
	}
	s.DnstapOutput = output.NewDnstapOutput(s, 0)
	return s, nil
}

type OutputFormat string

var (
	OutputFormatJsonV1 OutputFormat = "json/v1"
	OutputFormatGoTpl  OutputFormat = "go-template"
)

var _ types.OutputPlugin = &Output{}

// This is an experimental implementation.
// The file plug-in outputs the message to a file.
type Output struct {
	plugin.PluginCommon
	*output.DnstapOutput

	// File output config
	// see https://pkg.go.dev/gopkg.in/natefinch/lumberjack.v2#Logger
	Logger *lumberjack.Logger

	// File format
	Format OutputFormat

	// JSON Key Filter
	JSONIncludeKeys []string
	JSONExcludeKeys []string

	// Line go template for format type 'go-template"
	Template string

	t  *template.Template
	oc *types.OutputContext
}

func (f *Output) SetOutputContext(oc *types.OutputContext) {
	f.oc = oc
}

func (o *Output) Open() error {
	return nil
}

func (o *Output) Write(dm *types.DnstapMessage) error {
	switch o.Format {
	case OutputFormatJsonV1:
		buf, err := dm.ConvertV1JSONWithFilter(o.JSONIncludeKeys, o.JSONExcludeKeys)
		if err != nil {
			return err
		}
		if _, err := o.Logger.Write(buf); err != nil {
			return err
		}
		if _, err := o.Logger.Write([]byte("\n")); err != nil {
			return err
		}
	case OutputFormatGoTpl:
		data, err := dm.ConvertV1Flat()
		if err != nil {
			return err
		}
		if err := o.t.Execute(o.Logger, data); err != nil {
			return err
		}
		if _, err := o.Logger.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}

func (o *Output) Close() {
	o.Logger.Close()
}
