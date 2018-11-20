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
)

type DnstapOutput struct {
	handler   OutputHandler
	rbuf      *RBuf
	writeDone chan struct{}
}

func NewDnstapOutput(outputBufferSize uint, handler OutputHandler) *DnstapOutput {
	return &DnstapOutput{
		handler:   handler,
		rbuf:      NewRbuf(outputBufferSize),
		writeDone: make(chan struct{}),
	}
}
func (o *DnstapOutput) Run(ctx context.Context, errCh chan error) {
	for {
		if o.handler.open() != nil {
			continue
		}
		o.run(ctx, errCh)
	}
	o.handler.close()
	close(o.writeDone)
}
func (o *DnstapOutput) run(ctx context.Context, errCh chan error) {
	for {
		select {
		case <-ctx.Done():
			break
		default:
			if buf := o.rbuf.Read(); buf != nil {
				if err := o.handler.write(buf); err != nil {
					errCh <- err
					break
				}
			}
		}
	}
}

func (o *DnstapOutput) SetMessage(b []byte) {
	o.rbuf.Write(b)
}

func (o *DnstapOutput) WriteDone() <-chan struct{} {
	return o.writeDone
}
