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
	"time"

	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const MaxRetryDuration = time.Minute * 1

type OutputHandler interface {
	SetOutputContext(oc *types.OutputContext)
	Open() error
	Write(*types.DnstapMessage) error
	Close()
}

type DnstapOutput struct {
	handler        OutputHandler
	retryOpenCount uint
	maxRetry       uint
	oc             *types.OutputContext
}

func NewDnstapOutput(handler OutputHandler, maxRetry uint) *DnstapOutput {
	if handler == nil {
		panic("handler is nil")
	}
	return &DnstapOutput{
		handler:        handler,
		retryOpenCount: 0,
		maxRetry:       0,
	}
}

func (o *DnstapOutput) Start(ctx context.Context, oc *types.OutputContext) error {
	o.oc = oc
	o.handler.SetOutputContext(oc)
	o.oc.Logger.Debug("start output run")
L:
	for {
		select {
		case <-ctx.Done():
			o.oc.Logger.Debug("Run ctx done")
			break L
		default:
			if err := o.Run(ctx, oc.Reader); err != nil {
				if o.maxRetry != 0 && o.maxRetry <= o.retryOpenCount {
					return errors.Wrap(err, "failed to open output resource")
				}
				o.oc.Logger.Debug("output running error", zap.Error(err))
			}
		}
	}
	o.oc.Logger.Debug("end output run")
	return nil
}

func (o *DnstapOutput) Run(ctx context.Context, r types.Reader) error {
	if err := o.handler.Open(); err != nil {
		retryDuration := time.Second * time.Duration(1+o.retryOpenCount*o.retryOpenCount)
		if retryDuration > MaxRetryDuration {
			retryDuration = MaxRetryDuration
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(retryDuration):
		}
		o.retryOpenCount++
		return err
	}
	o.retryOpenCount = 0

	defer o.handler.Close()
	o.oc.Logger.Debug("start writer")
L:
	for {
		select {
		case <-ctx.Done():
			break L
		case frame := <-r.Read():
			if frame != nil {
				if err := o.handler.Write(frame); err != nil {
					o.oc.Logger.Debug("writer error", zap.Error(err))
					return err
				}
			}
		}
	}
	o.oc.Logger.Debug("end writer")
	return nil
}
