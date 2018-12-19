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

	log "github.com/sirupsen/logrus"
)

type DnstapOutput struct {
	handler OutputHandler
	rbuf    *RBuf
}

func NewDnstapOutput(outputBufferSize uint, handler OutputHandler) *DnstapOutput {
	return &DnstapOutput{
		handler: handler,
		rbuf:    NewRbuf(outputBufferSize, TotalRecvOutputFrame, TotalLostOutputFrame),
	}
}
func (o *DnstapOutput) Run(ctx context.Context) {
	var err error
L:
	for {
		select {
		case <-ctx.Done():
			log.Debug("Run ctx done")
			break L
		default:
			if err := o.handler.open(); err != nil {
				log.Debug(err)
				continue
			}
			childCtx, _ := context.WithCancel(ctx)
			if err = o.run(childCtx); err == nil {
				log.Debug(err)
				break L
			}
			log.Debug("run loop")
		}
	}
	log.Debug("close handle close")
	o.handler.close()
	log.Debug("output close")
	return
}
func (o *DnstapOutput) run(ctx context.Context) error {
L:
	for {
		select {
		case <-ctx.Done():
			break L
		case buf := <-o.rbuf.Read():
			if buf != nil {
				if err := o.handler.write(buf); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (o *DnstapOutput) SetMessage(b []byte) {
	o.rbuf.Write(b)
}
