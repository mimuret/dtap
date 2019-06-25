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

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type DnstapOutputParams struct {
	BufferSize  uint
	InCounter   prometheus.Counter
	LostCounter prometheus.Counter
	Handler     OutputHandler
}

type DnstapOutput struct {
	handler OutputHandler
	rbuf    *RBuf
}

func NewDnstapOutput(params *DnstapOutputParams) *DnstapOutput {
	return &DnstapOutput{
		handler: params.Handler,
		rbuf:    NewRbuf(params.BufferSize, params.InCounter, params.LostCounter),
	}
}

func (o *DnstapOutput) Run(ctx context.Context) {
	log.Debug("start output run")
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
			log.Debug("success open")
			childCtx, _ := context.WithCancel(ctx)
			err := o.run(childCtx)
			log.Debug("close handle close")
			o.handler.close()

			if err != nil {
				log.Debug(err)
			} else {
				break L
			}
		}
	}
	return
}
func (o *DnstapOutput) run(ctx context.Context) error {
	log.Debug("start writer")
L:
	for {
		select {
		case <-ctx.Done():
			break L
		case frame := <-o.rbuf.Read():
			if frame != nil {
				if err := o.handler.write(frame); err != nil {
					log.Debugf("writer error: %v", err)
					return err
				}
			}
		}
	}
	log.Debug("end writer")
	return nil
}

func (o *DnstapOutput) SetMessage(b []byte) {
	o.rbuf.Write(b)
}
