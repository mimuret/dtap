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
	"github.com/pkg/errors"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/farsightsec/golang-framestream"
	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/golang/protobuf/proto"
)

type DnstapFluentdOutput struct {
	config      *OutputFluentConfig
	fluetConfig fluent.Config
	enc         *framestream.Encoder
	logger      *fluent.Fluent
	handler     FluetOutput
}

func NewDnstapFluentdOutput(config *OutputFluentConfig, handler FluetOutput) *DnstapOutput {
	o := &DnstapFluentdOutput{
		config: config,
		fluetConfig: fluent.Config{
			FluentHost: config.GetHost(),
			FluentPort: config.GetPort(),
			Async:      false},
		handler: handler,
	}
	return NewDnstapOutput(config.GetBufferSize(), o)
}

func (o *DnstapFluentdOutput) open() error {
	var err error
	o.logger, err = fluent.New(o.fluetConfig)
	if err != nil {
		return errors.Wrapf(err, "can't create fluent logger")
	}

	return nil
}

func (o *DnstapFluentdOutput) write(frame []byte) error {
	dt := &dnstap.Dnstap{}
	if err := proto.Unmarshal(frame, dt); err != nil {
		return err
	}
	o.handler.handle(o.logger, dt)
	return nil
}

func (o *DnstapFluentdOutput) close() {
	o.logger.Close()
}
