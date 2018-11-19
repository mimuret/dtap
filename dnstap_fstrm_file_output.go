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
	"context"
	"io"
	"os"
	"strings"
	"time"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"
	strftime "github.com/jehiah/go-strftime"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/ulikunitz/xz"
)

type DnstapFstrmFileOutput struct {
	config          *OutputFileConfig
	currentFilename string
	outputChannel   chan []byte
	enc             *framestream.Encoder
	finished        bool
}

func NewDnstapFstrmFileOutput(config *OutputFileConfig) *DnstapFstrmFileOutput {
	return &DnstapFstrmFileOutput{
		config:        config,
		outputChannel: make(chan []byte, config.GetChannelSize()),
	}
}

func (o *DnstapFstrmFileOutput) newFile() error {
	var w io.Writer
	filename := strftime.Format(o.config.GetPath(), time.Now())
	f, err := os.Create(filename)
	if err != nil {
		return errors.Wrapf(err, "can't create file %s", filename)
	}
	if strings.HasSuffix(filename, "gz") {
		w = gzip.NewWriter(f)
	} else if strings.HasSuffix(filename, "xz") {
		w, err = xz.NewWriter(f)
		if err != nil {
			return errors.Wrapf(err, "can't create xz wirter file %s", filename)
		}
	} else {
		w = f
	}

	o.enc, err = framestream.NewEncoder(w, &framestream.EncoderOptions{ContentType: dnstap.FSContentType, Bidirectional: false})
	if err != nil {
		return errors.Wrapf(err, "can't create framestream encorder %s", filename)
	}
	o.currentFilename = filename
	return nil
}

func (o *DnstapFstrmFileOutput) runWrite(ctx context.Context, errCh chan error) bool {
	ticker := time.NewTicker(FlushTimeout)
	for {
		select {
		case <-ctx.Done():
			break
		case <-ticker.C:
			o.enc.Flush()
			filename := strftime.Format(o.config.GetPath(), time.Now())
			if filename != o.currentFilename {
				o.enc.Close()
				return true
			}
		case frame := <-o.outputChannel:
			if _, err := o.enc.Write(frame); err != nil {
				if log.GetLevel() >= log.DebugLevel {
					errCh <- errors.Wrap(err, "can't write frame")
				}
				o.enc.Flush()
				o.enc.Close()
				return true
			}
		}
	}
	return false
}

func (o *DnstapFstrmFileOutput) runOpen(ctx context.Context, errCh chan error) bool {
	ticker := time.NewTicker(FlushTimeout)
	for {
		select {
		case <-ctx.Done():
			break
		case <-ticker.C:
			if err := o.newFile(); err != nil {
				if log.GetLevel() >= log.WarnLevel {
					errCh <- errors.Wrap(err, "can't create file")
				}
			} else {
				if log.GetLevel() >= log.DebugLevel {
					errCh <- errors.New("create file scussesful")
				}
				return true
			}
		}
	}
	return false
}

func (o *DnstapFstrmFileOutput) Run(ctx context.Context, errCh chan error) {
	for {
		if log.GetLevel() >= log.DebugLevel {
			errCh <- errors.New("start runOpen")
		}
		if !o.runOpen(ctx, errCh) {
			break
		}
		if log.GetLevel() >= log.DebugLevel {
			errCh <- errors.New("start runWrite")
		}
		if !o.runWrite(ctx, errCh) {
			break
		}
	}
	o.enc.Flush()
	o.enc.Close()
	o.finished = true
}

func (o *DnstapFstrmFileOutput) GetOutputChannel() chan []byte {
	return o.outputChannel
}

func (o *DnstapFstrmFileOutput) Finished() bool {
	return o.finished
}
