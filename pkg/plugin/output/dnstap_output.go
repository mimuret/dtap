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

	"github.com/mimuret/dtap/v2/pkg/logger"
	"github.com/mimuret/dtap/v2/pkg/types"
	"go.uber.org/zap"
)

type OutputHandler interface {
	Open() error
	Write(*types.DnstapMessage) error
	Close()
}

type DnstapOutput struct {
	handler        OutputHandler
	logger         *zap.Logger
	retryOpenCount uint
}

func NewDnstapOutput(handler OutputHandler) *DnstapOutput {
	if handler == nil {
		panic("handler is nil")
	}
	return &DnstapOutput{
		handler:        handler,
		logger:         logger.GetLogger(),
		retryOpenCount: 0,
	}
}

func (o *DnstapOutput) Start(ctx context.Context, r types.Reader) error {
	o.logger.Debug("start output run")
L:
	for {
		select {
		case <-ctx.Done():
			o.logger.Debug("Run ctx done")
			break L
		default:
			if err := o.Run(ctx, r); err != nil {
				o.logger.Debug("output running error", zap.Error(err))
			}
		}
	}
	o.logger.Debug("end output run")
	return nil
}

func (o *DnstapOutput) Run(ctx context.Context, r types.Reader) error {
	if err := o.handler.Open(); err != nil {
		retryDuration := time.Second * time.Duration(1+o.retryOpenCount*o.retryOpenCount)
		if retryDuration > time.Minute*3 {
			retryDuration = time.Minute * 3
		}
		time.Sleep(retryDuration)
		o.retryOpenCount++
		return err
	}
	o.retryOpenCount = 0

	defer o.handler.Close()
	o.logger.Debug("start writer")
L:
	for {
		select {
		case <-ctx.Done():
			break L
		case frame := <-r.Read():
			if frame != nil {
				if err := o.handler.Write(frame); err != nil {
					o.logger.Debug("writer error", zap.Error(err))
					return err
				}
			}
		}
	}
	o.logger.Debug("end writer")
	return nil
}
