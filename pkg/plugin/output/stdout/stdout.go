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
	"context"
	"io"
	"math"
	"os"
	"text/template"

	"errors"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/plugin/output"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/mimuret/dtap/v3/pkg/types"
)

const PLUGIN_NAME = "stdout"

const DefaultTemplate = `{{ .Message.Timestamp }} {{ .Message.Type }} {{ .Message.Qclass }} {{ .Message.Qtype }} {{ .Message.Qname }}`

func init() {
	_ = registry.RegisterOutputPlugin(PLUGIN_NAME, Setup)
}

func Setup(cfg *config.OutputBlock) (types.OutputPlugin, error) {
	var err error
	p := &Stdout{
		OutputBlock:   *cfg,
		Format:        OutputFormatJsonV1,
		OutputFilters: &types.OutputFilters{},
		Template:      DefaultTemplate,
	}
	diags := gohcl.DecodeBody(cfg.Body, nil, p)
	if diags.HasErrors() {
		return nil, plugin.PluginError(p, "failed to setup stdout plugin: %w", errors.Join(diags.Errs()...))
	}
	switch p.Format {
	case OutputFormatGoTpl:
		p.t, err = template.New("").Parse(p.Template)
		if err != nil {
			return nil, plugin.PluginError(p, "template is an invalid value: %w", err)
		}
	case OutputFormatJsonV1:
	default:
		return nil, plugin.PluginError(p, "format is an invalid value: %s", p.Format)
	}
	p.DnstapOutput = output.NewDnstapOutput(p, uint(0))

	if p.Stderr {
		p.w = os.Stderr
	} else {
		p.w = os.Stdout
	}
	return p, nil
}

type OutputFormat string

var (
	OutputFormatJsonV1 OutputFormat = "json/v1"
	OutputFormatGoTpl  OutputFormat = "go-template"
)

var _ types.OutputPlugin = &Stdout{}

// The stdout plugin ouput the message to stdout.
type Stdout struct {
	config.OutputBlock
	*output.DnstapOutput

	// Stderr is true, output to stderr instead of stdout.
	Stderr bool `hcl:"stderr,optional"`

	// File format
	Format OutputFormat `hcl:"format,optional"`

	// JSON Key Filter
	OutputFilters *types.OutputFilters `hcl:"output_filters,block"`

	// Line go template for format type 'go-template"
	Template string `hcl:"template,optional"`

	t *template.Template

	w io.Writer
}

func (p *Stdout) Open(context.Context) error {
	return nil
}

func (p *Stdout) Write(ctx context.Context, dm *types.DnstapMessage) error {
	switch p.Format {
	case OutputFormatJsonV1:
		buf, err := dm.ConvertV1JSONWithFilter(*p.OutputFilters)
		if err != nil {
			return err
		}
		if _, err := p.w.Write(buf); err != nil {
			return err
		}
		if _, err := p.w.Write([]byte("\n")); err != nil {
			return err
		}
	case OutputFormatGoTpl:
		val, err := types.CreateMsgValue(p, dm)
		if err != nil {
			return err
		}
		if err := p.t.Execute(p.w, val); err != nil {
			return err
		}
		if _, err := p.w.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}

func (p *Stdout) Close(context.Context) {

}

func (p *Stdout) MaxConcurrent() uint {
	return math.MaxUint32
}
