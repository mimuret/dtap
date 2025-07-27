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
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"
	"github.com/mimuret/dtap/v3/pkg/types"
	"github.com/pkg/errors"
)

type SocketOutput interface {
	types.OutputPlugin
	NewConnect(context.Context) (io.Writer, error)
	Close(context.Context)
}

var _ OutputHandler = &DnstapFstrmSocketOutput{}

type DnstapFstrmSocketOutput struct {
	handler      SocketOutput
	flushTimeout time.Duration

	enc     *framestream.Encoder
	encOpt  *framestream.EncoderOptions
	closeCh chan struct{}
	wg      *sync.WaitGroup
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

func (o *DnstapFstrmSocketOutput) Open(ctx context.Context) error {
	w, err := o.handler.NewConnect(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect socket: %w", err)
	}
	o.enc, err = framestream.NewEncoder(w, o.encOpt)
	if err != nil {
		o.handler.Close(ctx)
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
				if err := o.enc.Flush(); err != nil {
					return
				}
			}
		}
	}()
	return nil
}

func (o *DnstapFstrmSocketOutput) Write(ctx context.Context, dm *types.DnstapMessage) error {
	if _, err := o.enc.Write(dm.GetRaw()); err != nil {
		return err
	}
	return nil
}

func (o *DnstapFstrmSocketOutput) Close(ctx context.Context) {
	// stop flush loop
	close(o.closeCh)
	o.wg.Wait()

	// close fstrm
	o.enc.Close()

	// close connection
	o.handler.Close(ctx)
}
