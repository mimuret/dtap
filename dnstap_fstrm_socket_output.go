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
	"context"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	framestream "github.com/farsightsec/golang-framestream"
)

type DnstapFstrmSocketOutput struct {
	handler       SocketOutput
	outputChannel chan []byte
	enc           *framestream.Encoder
	finished      bool
}

func NewDnstapFstrmSocketOutput(outputChannelSize int, handler SocketOutput) *DnstapFstrmSocketOutput {
	return &DnstapFstrmSocketOutput{
		handler:       handler,
		outputChannel: make(chan []byte, outputChannelSize),
	}
}
func (o *DnstapFstrmSocketOutput) runWrite(ctx context.Context, errCh chan error) bool {
	ticker := time.NewTicker(FlushTimeout)
	for {
		select {
		case <-ctx.Done():
			break
		case <-ticker.C:
			o.enc.Flush()
		case frame := <-o.outputChannel:
			if _, err := o.enc.Write(frame); err != nil {
				o.enc.Flush()
				o.enc.Close()
				return true
			}
		}
	}
	return false
}

func (o *DnstapFstrmSocketOutput) runOpen(ctx context.Context, errCh chan error) bool {
	ticker := time.NewTicker(FlushTimeout)
	for {
		select {
		case <-ticker.C:
			if enc, err := o.handler.newConnect(); err != nil {
				if log.GetLevel() >= log.InfoLevel {

					errCh <- errors.Wrapf(err, "can't connect socket")
				}
			} else {
				o.enc = enc
			}
		case <-ctx.Done():
			return true
			break
		}
	}
	return false
}

func (o *DnstapFstrmSocketOutput) Run(ctx context.Context, errCh chan error) {
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
	close(o.outputChannel)
	o.enc.Flush()
	o.enc.Close()
	o.finished = true
}

func (o *DnstapFstrmSocketOutput) GetOutputChannel() chan []byte {
	return o.outputChannel
}

func (o *DnstapFstrmSocketOutput) Finished() bool {
	return o.finished
}
