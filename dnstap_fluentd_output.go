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

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/farsightsec/golang-framestream"
	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/golang/protobuf/proto"
)

type DnstapFluentdOutput struct {
	config        *OutputFluentConfig
	outputChannel chan []byte
	enc           *framestream.Encoder
	logger        *fluent.Fluent
	finished      bool
	handler       FluetOutput
}

func NewDnstapFluentdOutput(config *OutputFluentConfig, handler FluetOutput) (*DnstapFluentdOutput, error) {
	var err error
	o := &DnstapFluentdOutput{
		config:        config,
		outputChannel: make(chan []byte, config.GetChannelSize()),
		handler:       handler,
	}
	o.logger, err = fluent.New(fluent.Config{
		FluentHost: config.GetHost(),
		FluentPort: config.GetPort(),
		Async:      false},
	)
	if err != nil {
		return nil, errors.Wrapf(err, "can't create fluent logger")
	}
	return o, nil
}

func (o *DnstapFluentdOutput) handle(frame []byte, errCh chan error) {
	dt := &dnstap.Dnstap{}
	if err := proto.Unmarshal(frame, dt); err != nil {
		if log.GetLevel() >= log.DebugLevel {
			errCh <- errors.Wrapf(err, "proto.Unmarshal() failed: %s: %s\n", err)
		}
		return
	}
	o.handler.handle(o.logger, dt, errCh)
}

func (o *DnstapFluentdOutput) Run(ctx context.Context, errCh chan error) {
	for {
		select {
		case frame := <-o.outputChannel:
			o.handle(frame, errCh)
		case <-ctx.Done():
			break
		}
	}
	o.logger.Close()
	o.finished = true
}

func (o *DnstapFluentdOutput) GetOutputChannel() chan []byte {
	return o.outputChannel
}

func (o *DnstapFluentdOutput) Finished() bool {
	return o.finished
}
