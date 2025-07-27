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
	"bytes"
	"context"
	"os"
	"sync"
	"text/template"
	"time"

	"errors"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/lestrrat-go/strftime"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/plugin/output"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/mimuret/dtap/v3/pkg/types"
	"github.com/spf13/afero"
	"go.uber.org/zap"
)

var fs afero.Fs

const PLUGIN_NAME = "file"

const DefaultTemplate = `{{ .Message.Timestamp }} {{ .Message.Type }} {{ .Message.Qclass }} {{ .Message.Qtype }} {{ .Message.Qname }}`

func init() {
	fs = afero.NewOsFs()
	_ = registry.RegisterOutputPlugin(PLUGIN_NAME, Setup)
}

func Setup(cfg *config.OutputBlock) (types.OutputPlugin, error) {
	var err error
	p := &Output{
		OutputBlock:   *cfg,
		Format:        OutputFormatJsonV1,
		Permission:    0644,
		OutputFilters: &types.OutputFilters{},
	}
	diags := gohcl.DecodeBody(cfg.Body, nil, p)
	if diags.HasErrors() {
		return nil, plugin.PluginError(p, "failed to setup file plugin: %w", errors.Join(diags.Errs()...))
	}
	switch p.Format {
	case OutputFormatGoTpl:
		if p.Template == "" {
			p.Template = DefaultTemplate
		}
		p.t, err = template.New("").Parse(p.Template)
		if err != nil {
			return nil, plugin.PluginError(p, "template is invalid: %w", err)
		}
	case OutputFormatJsonV1:
	default:
		return nil, plugin.PluginError(p, "format is invalid: %s", p.Format)
	}
	p.DnstapOutput = output.NewDnstapOutput(p, 0)
	p.w, err = NewWriter(p.Path, p.Permission)
	if err != nil {
		return nil, plugin.PluginError(p, "failed to create writer: %w", err)
	}
	return p, nil
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
	config.OutputBlock
	*output.DnstapOutput

	// Output file path
	Path string `hcl:"path"`

	// Output file permission
	Permission os.FileMode `hcl:"permission,optional"`

	// File format
	Format OutputFormat `hcl:"format,optional"`

	// JSON Key Filter
	OutputFilters *types.OutputFilters `hcl:"output_filters,block"`

	// Line go template for format type 'go-template"
	Template string `hcl:"template,optional"`

	t *template.Template
	w *writer
}

func (o *Output) Write(ctx context.Context, dm *types.DnstapMessage) error {
	switch o.Format {
	case OutputFormatJsonV1:
		buf, err := dm.ConvertV1JSONWithFilter(*o.OutputFilters)
		if err != nil {
			return err
		}
		if _, err := o.w.Write(buf); err != nil {
			return err
		}
		if _, err := o.w.Write([]byte("\n")); err != nil {
			return err
		}
	case OutputFormatGoTpl:
		val, err := types.CreateMsgValue(o, dm)
		if err != nil {
			return err
		}
		if err := o.t.Execute(o.w, val); err != nil {
			return err
		}
		if _, err := o.w.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}

func (o *Output) Open(ctx context.Context) error {
	return o.w.Open()
}

func (o *Output) Close(ctx context.Context) {
	if err := o.w.Close(); err != nil {
		ctxzap.Error(ctx, "failed to close file writer", zap.Error(err))
	}
}

func (p *Output) MaxConcurrent() uint {
	return uint(1)
}

type writer struct {
	mu         sync.Mutex
	path       *strftime.Strftime
	fileName   string
	permission os.FileMode
	f          afero.File
}

func NewWriter(path string, permission os.FileMode) (*writer, error) {
	strftimeStr, err := strftime.New(path)
	if err != nil {
		return nil, err
	}

	w := &writer{path: strftimeStr, permission: permission}
	if _, err := w.getFilePath(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *writer) checkFilePath() bool {
	currentFile, err := w.getFilePath()
	if err != nil {
		// エラーが発生した場合はファイルパスが変更されていないと仮定
		return false
	}
	return w.fileName != currentFile
}

func (w *writer) getFilePath() (string, error) {
	buf := bytes.NewBuffer(nil)
	if err := w.path.Format(buf, time.Now()); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (w *writer) Write(b []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.checkFilePath() || w.f == nil {
		if err := w.Open(); err != nil {
			return 0, err
		}
	}
	return w.f.Write(b)
}

func (w *writer) Open() error {
	// 既存のファイルハンドルがある場合は先にクローズ
	if w.f != nil {
		if closeErr := w.f.Close(); closeErr != nil {
			// ログに記録するか、エラーを返すかを検討
			_ = closeErr
		}
		w.f = nil
	}

	fileName, err := w.getFilePath()
	if err != nil {
		return err
	}

	f, err := fs.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, w.permission)
	if err != nil {
		return err
	}

	w.fileName = fileName
	w.f = f
	return nil
}

func (w *writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.f != nil {
		if err := w.f.Close(); err != nil {
			return err
		}
		w.f = nil
	}
	return nil
}
