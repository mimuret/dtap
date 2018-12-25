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
	"compress/gzip"
	"io"
	"os"
	"strings"
	"time"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"
	strftime "github.com/jehiah/go-strftime"
	"github.com/pkg/errors"
	"github.com/ulikunitz/xz"
)

type DnstapFstrmFileOutput struct {
	config          *OutputFileConfig
	currentFilename string
	enc             *framestream.Encoder
	writer          io.WriteCloser
	opened          chan bool
}

func NewDnstapFstrmFileOutput(config *OutputFileConfig) *DnstapOutput {
	file := &DnstapFstrmFileOutput{
		config: config,
	}
	return NewDnstapOutput(config.GetBufferSize(), file)
}

func (o *DnstapFstrmFileOutput) open() error {
	filename := strftime.Format(o.config.GetPath(), time.Now())
	f, err := os.Create(filename)
	if err != nil {
		return errors.Wrapf(err, "can't create file %s", filename)
	}
	if strings.HasSuffix(filename, "gz") {
		o.writer = gzip.NewWriter(f)
	} else if strings.HasSuffix(filename, "xz") {
		o.writer, err = xz.NewWriter(f)
		if err != nil {
			return errors.Wrapf(err, "can't create xz wirter file %s", filename)
		}
	} else {
		o.writer = f
	}

	o.enc, err = framestream.NewEncoder(o.writer, &framestream.EncoderOptions{ContentType: dnstap.FSContentType, Bidirectional: false})
	if err != nil {
		return errors.Wrapf(err, "can't create framestream encorder %s", filename)
	}
	o.currentFilename = filename
	o.opened = make(chan bool)
	go func() {
		ticker := time.NewTicker(FlushTimeout)
		for {
			select {
			case <-o.opened:
				return
			case <-ticker.C:
				o.enc.Flush()
				filename := strftime.Format(o.config.GetPath(), time.Now())
				if filename != o.currentFilename {
					o.enc.Close()
					o.writer.Close()
					ticker.Stop()
				}
			}
		}
	}()
	return nil
}

func (o *DnstapFstrmFileOutput) write(frame []byte) error {
	o.enc.Write(frame)
	return nil
}

func (o *DnstapFstrmFileOutput) close() {
	o.enc.Flush()
	o.enc.Close()
	o.writer.Close()
	close(o.opened)
}
