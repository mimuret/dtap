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
	"bufio"
	"text/template"
	"time"

	json "github.com/goccy/go-json"
	"github.com/lestrrat-go/strftime"
	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/output"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
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
	if s.Logger == nil {
		return nil, errors.New("missing parameter Logger")
	}
	if s.Logger.Filename == "" {
		return nil, errors.New("missing parameter Logger.Filename")
	}
	testTime := time.Now()
	if _, err := strftime.Format(s.Logger.Filename, testTime); err != nil {
		return nil, errors.Wrap(err, "Logger.Filename contains invalid strftime format")
	}
	timeFormat := s.Logger.FilenameTimeFormat
	if timeFormat == "" {
		timeFormat = defaultFilenameTimeFormat
	}
	if _, err := strftime.Format(timeFormat, testTime); err != nil {
		return nil, errors.Wrap(err, "Logger.FilenameTimeFormat contains invalid strftime format")
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

type CompressType string

const (
	CompressTypeGzip CompressType = "gzip"
	CompressTypeZstd CompressType = "zstd"
)

type Logger struct {
	Filename   string `json:"filename" yaml:"filename"`
	MaxSize    int    `json:"maxsize" yaml:"maxsize"`
	MaxAge     int    `json:"maxage" yaml:"maxage"`
	MaxBackups int    `json:"maxbackups" yaml:"maxbackups"`
	LocalTime  bool   `json:"localtime" yaml:"localtime"`
	Compress   bool   `json:"compress" yaml:"compress"`

	CompressType    CompressType `json:"compress_type" yaml:"compress_type"`
	CompressWorkers int          `json:"compress_workers" yaml:"compress_workers"`

	// ローテーション後のファイル名に付与する日時フォーマット (strftime 形式)
	// デフォルト: "%Y-%m-%dT%H-%M-%S"
	FilenameTimeFormat string `json:"filename_time_format" yaml:"filename_time_format"`
}

// This is an experimental implementation.
// The file plug-in outputs the message to a file.
type Output struct {
	plugin.PluginCommon
	*output.DnstapOutput

	// File output config
	Logger *Logger

	// File format
	Format OutputFormat

	// JSON Key Filter
	OutputFilters types.OutputFilters

	// Line go template for format type 'go-template"
	Template string

	t  *template.Template
	oc *types.OutputContext
	rw *rotatingWriter
	w  *bufio.Writer
}

func (f *Output) SetOutputContext(oc *types.OutputContext) {
	f.oc = oc
}

func (o *Output) Open() error {
	rw, err := newRotatingWriter(o.Logger)
	if err != nil {
		return err
	}
	o.rw = rw
	o.w = bufio.NewWriterSize(rw, 256*1024)
	return nil
}

func (o *Output) Write(dm *types.DnstapMessage) error {
	switch o.Format {
	case OutputFormatJsonV1:
		buf, err := dm.ConvertV1JSONWithFilter(o.OutputFilters)
		if err != nil {
			return err
		}
		if _, err := o.w.Write(buf); err != nil {
			return err
		}
		if err := o.w.WriteByte('\n'); err != nil {
			return err
		}
	case OutputFormatGoTpl:
		data, err := dm.ConvertV1Flat()
		if err != nil {
			return err
		}
		if err := o.t.Execute(o.w, data); err != nil {
			return err
		}
		if err := o.w.WriteByte('\n'); err != nil {
			return err
		}
	}
	return o.w.Flush()
}

func (o *Output) Close() {
	if o.w != nil {
		o.w.Flush()
	}
	if o.rw != nil {
		o.rw.Close()
	}
}
