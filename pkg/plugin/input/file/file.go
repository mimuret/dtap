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
package file

import (
	"context"

	"errors"

	framestream "github.com/farsightsec/golang-framestream"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/plugin/input"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/mimuret/dtap/v3/pkg/types"
	"github.com/spf13/afero"
)

const PLUGIN_NAME = "file"

func init() {
	_ = registry.RegisterInputPlugin(PLUGIN_NAME, Setup)
}

func Setup(cfg *config.InputBlock) (types.InputPlugin, error) {
	p := &File{
		InputBlock: *cfg,
		Format:     input.FormatDNSTAP,
	}
	// Decode the HCL body into the file struct.
	diags := gohcl.DecodeBody(cfg.Body, nil, p)
	if diags.HasErrors() {
		return nil, plugin.PluginError(p, "failed to setup file plugin: %w", errors.Join(diags.Errs()...))
	}
	if p.Path == "" {
		return nil, plugin.PluginError(p, "missing parameter Path")
	}
	p.is = input.NewInputServer(p, p.Format, &framestream.DecoderOptions{Bidirectional: false})
	if p.is == nil {
		return nil, plugin.PluginError(p, "invalid format")
	}

	p.fs = afero.NewOsFs()
	return p, nil
}

var _ types.InputPlugin = &File{}

// The file plugin enters the DNSTAP message only once from the file.
// Example configuration:
// ```hcl
//
//	input "file" "example" {
//	  path = "/var/log/dnstap.log"
//	  format = "DNSTAP"
//	}
//
// ```
type File struct {
	config.InputBlock

	// Message format default is "DNSTAP".
	// Supported values are "DNSTAP", "DtapFrame",
	Format string `hcl:"format,optional"`

	// File Path
	Path string `hcl:"path"`

	fs afero.Fs

	is *input.InputServer
}

func (p *File) Start(ctx context.Context, forwarder types.Forwarder) error {
	r, err := p.fs.Open(p.Path)
	if err != nil {
		return plugin.PluginError(p, "failed to open file: %w", err)
	}
	if err := p.is.Read(ctx, forwarder, r); err != nil {
		return plugin.PluginError(p, "failed to push message: %w", err)
	}
	return nil
}
