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
	"time"

	framestream "github.com/farsightsec/golang-framestream"
	"github.com/pkg/errors"
)

type DnstapFstrmSocketOutput struct {
	handler SocketOutput
	enc     *framestream.Encoder
	opened  chan bool
}

func NewDnstapFstrmSocketOutput(handler SocketOutput, params *DnstapOutputParams) *DnstapOutput {
	params.Handler = &DnstapFstrmSocketOutput{
		handler: handler,
	}
	return NewDnstapOutput(params)
}

func (o *DnstapFstrmSocketOutput) open() error {
	var err error
	if o.enc, err = o.handler.newConnect(); err != nil {
		return errors.Wrapf(err, "can't connect socket")
	}
	o.opened = make(chan bool)
	go func() {
		ticker := time.NewTicker(FlushTimeout)
		for {
			select {
			case <-o.opened:
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

func (o *DnstapFstrmSocketOutput) write(frame []byte) error {
	if _, err := o.enc.Write(frame); err != nil {
		o.close()
		return err
	}
	return nil
}

func (o *DnstapFstrmSocketOutput) close() {
	o.enc.Flush()
	o.enc.Close()
	close(o.opened)
}
