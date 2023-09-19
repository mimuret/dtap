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
	"os"
	"text/template"

	json "github.com/goccy/go-json"
	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/output"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
)

func init() {
	_ = registry.RegisterOutputPlugin("stdout", setup)
}

func setup(bs json.RawMessage) (types.OutputPlugin, error) {
	var err error
	s := &Stdout{
		Type: OutputFormatJsonV1,
	}
	if err := json.Unmarshal(bs, s); err != nil {
		return nil, errors.Wrap(err, "failed to decode config")
	}
	switch s.Type {
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
	s.DnstapOutput = output.NewDnstapOutput(s, s.MaxRetry)
	return s, nil
}

type OutputFormat string

var (
	OutputFormatJsonV1 OutputFormat = "json/v1"
	OutputFormatGoTpl  OutputFormat = "go-template"
)

var _ types.OutputPlugin = &Stdout{}

// The stdout plugin ouput the message to stdout.
type Stdout struct {
	plugin.PluginCommon
	*output.DnstapOutput

	// output format type
	Type OutputFormat

	// Line go template for format type 'go-template"
	Template string

	t  *template.Template
	oc *types.OutputContext
}

func (f *Stdout) SetOutputContext(oc *types.OutputContext) {
	f.oc = oc
}

func (o *Stdout) Open() error {
	return nil
}

func (o *Stdout) Write(dm *types.DnstapMessage) error {
	switch o.Type {
	case OutputFormatJsonV1:
		buf, err := dm.ConvertV1JSON()
		if err != nil {
			return err
		}
		if _, err := os.Stdout.Write(buf); err != nil {
			return err
		}
		if _, err := os.Stdout.Write([]byte("\n")); err != nil {
			return err
		}
	case OutputFormatGoTpl:
		data, err := dm.ConvertV1Flat()
		if err != nil {
			return err
		}
		if err := o.t.Execute(os.Stdout, data); err != nil {
			return err
		}
		if _, err := os.Stdout.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}

func (o *Stdout) Close() {

}
