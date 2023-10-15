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
	"fmt"

	"github.com/goccy/go-json"

	framestream "github.com/farsightsec/golang-framestream"
	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/input"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func init() {
	_ = registry.RegisterInputPlugin("file", SetupFile)
}

func SetupFile(bs json.RawMessage) (types.InputPlugin, error) {
	p := &File{
		FormatMeta: input.FormatMeta{
			Format: input.FormatDNSTAP,
		},
	}
	if err := json.Unmarshal(bs, p); err != nil {
		return nil, errors.Wrapf(err, "failed to decode config")
	}
	if p.Path == "" {
		return nil, errors.New("missing parameter Path")
	}
	p.is = input.NewInputServer(p, &framestream.DecoderOptions{Bidirectional: false})
	if p.is == nil {
		return nil, errors.Errorf("invalid format")
	}

	p.fs = afero.NewOsFs()
	return p, nil
}

var _ types.InputPlugin = &File{}

// The file plugin enters the DNSTAP message only once from the file.
type File struct {
	plugin.PluginCommon

	fs afero.Fs

	// File Path
	Path string
	// File format
	input.FormatMeta

	is *input.InputServer
}

func (p *File) Start(_ context.Context, ic *types.InputContext) error {
	r, err := p.fs.Open(p.Path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	if err := p.is.Read(r, ic.Writer, ic); err != nil {
		return fmt.Errorf("failed to push message: %w", err)
	}
	return nil
}
