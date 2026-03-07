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

package output

import (
	"io"
	"sync"
	"time"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
)

type SocketOutput interface {
	SetOutputContext(*types.OutputContext)
	NewConnect() (io.Writer, error)
	Close()
}

var _ OutputHandler = &DnstapFstrmSocketOutput{}

type DnstapFstrmSocketOutput struct {
	handler      SocketOutput
	flushTimeout time.Duration
	oc           *types.OutputContext

	enc     *framestream.Encoder
	encOpt  *framestream.EncoderOptions
	closeCh chan struct{}
	wg      *sync.WaitGroup
	mu      sync.Mutex
}

func NewDnstapFstrmSocketOutput(handler SocketOutput, flushTimeout time.Duration, encOpt *framestream.EncoderOptions) *DnstapFstrmSocketOutput {
	if encOpt == nil {
		encOpt = &framestream.EncoderOptions{
			ContentType:   dnstap.FSContentType,
			Bidirectional: true,
			Timeout:       time.Second,
		}
	}
	return &DnstapFstrmSocketOutput{
		encOpt:       encOpt,
		handler:      handler,
		flushTimeout: flushTimeout,
		wg:           &sync.WaitGroup{},
	}
}

func (o *DnstapFstrmSocketOutput) SetOutputContext(oc *types.OutputContext) {
	o.oc = oc
	o.handler.SetOutputContext(oc)
}

func (o *DnstapFstrmSocketOutput) Open() error {
	w, err := o.handler.NewConnect()
	if err != nil {
		return errors.Wrap(err, "failed to connect socket")
	}
	o.enc, err = framestream.NewEncoder(w, o.encOpt)
	if err != nil {
		o.handler.Close()
		return errors.Wrapf(err, "failed to create fstrm encoder")
	}
	o.closeCh = make(chan struct{})
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		ticker := time.NewTicker(o.flushTimeout)
		for {
			select {
			case <-o.closeCh:
				return
			case <-ticker.C:
				o.mu.Lock()
				err := o.enc.Flush()
				o.mu.Unlock()
				if err != nil {
					return
				}
			}
		}
	}()
	return nil
}

func (o *DnstapFstrmSocketOutput) Write(dm *types.DnstapMessage) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if _, err := o.enc.Write(dm.GetRaw()); err != nil {
		return err
	}
	return nil
}

func (o *DnstapFstrmSocketOutput) Close() {
	// stop flush loop
	close(o.closeCh)
	o.wg.Wait()

	// close fstrm
	o.mu.Lock()
	o.enc.Close()
	o.mu.Unlock()

	// close connection
	o.handler.Close()
}
