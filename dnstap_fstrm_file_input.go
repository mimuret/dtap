/*
 * Copyright (c) 2018 Manabu Sonoda
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

package dtap

import (
	"compress/bzip2"
	"compress/gzip"
	"context"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/ulikunitz/xz"
)

type DnstapFstrmFileInput struct {
	config   *InputFileConfig
	input    *DnstapFstrmInput
	readDone chan struct{}
}

type DnstapFstrmFileReadCloser struct {
	reader io.Reader
	file   *os.File
}

func NewDnstapFstrmFileReadCloser(r io.Reader, f *os.File) *DnstapFstrmFileReadCloser {
	return &DnstapFstrmFileReadCloser{
		reader: r,
		file:   f,
	}
}
func (rc *DnstapFstrmFileReadCloser) Read(p []byte) (int, error) {
	return rc.reader.Read(p)
}
func (rc *DnstapFstrmFileReadCloser) Close() error {
	return rc.file.Close()
}

func NewDnstapFstrmFileInput(config *InputFileConfig) (*DnstapFstrmFileInput, error) {
	var r io.ReadCloser
	f, err := os.Open(config.GetPath())
	if err != nil {
		return nil, errors.Wrapf(err, "watch failed, path: %s", config.GetPath())
	}

	if strings.HasSuffix(config.GetPath(), "gz") {
		r, err = gzip.NewReader(f)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create gzip reader, path: %s", config.GetPath())
		}
	} else if strings.HasSuffix(config.GetPath(), "bz2") {
		cmp := bzip2.NewReader(f)
		r = NewDnstapFstrmFileReadCloser(cmp, f)
	} else if strings.HasSuffix(config.GetPath(), "xz") {
		cmp, err := xz.NewReader(f)
		r = NewDnstapFstrmFileReadCloser(cmp, f)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create xz reader, path: %s", config.GetPath())
		}
	} else {
		r = f
	}
	input, err := NewDnstapFstrmInput(r, false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create fstrm input, path: %s", config.GetPath())
	}

	i := &DnstapFstrmFileInput{
		config:   config,
		input:    input,
		readDone: make(chan struct{}),
	}
	return i, nil
}

func (i *DnstapFstrmFileInput) Run(ctx context.Context, rbuf *RBuf) error {
	childCtx, _ := context.WithCancel(ctx)
	err := i.input.Read(childCtx, rbuf)
	close(i.readDone)
	return err
}

func (i *DnstapFstrmFileInput) ReadDone() <-chan struct{} {
	return i.readDone
}
