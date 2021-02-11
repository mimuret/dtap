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
	"fmt"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"

	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/golang/protobuf/proto"
)

type DnstapFluentdOutput struct {
	config      *OutputFluentConfig
	fluetConfig fluent.Config
	enc         *framestream.Encoder
	client      *fluent.Fluent
	flatOption  DnstapFlatOption
	tag         string
}

func NewDnstapFluentdOutput(config *OutputFluentConfig, params *DnstapOutputParams) *DnstapOutput {
	params.Handler = &DnstapFluentdOutput{
		config:     config,
		flatOption: &config.Flat,
		fluetConfig: fluent.Config{
			FluentHost: config.GetHost(),
			FluentPort: config.GetPort(),
			Async:      false},
		tag: config.GetTag(),
	}

	return NewDnstapOutput(params)
}

func (o *DnstapFluentdOutput) open() error {
	var err error
	o.client, err = fluent.New(o.fluetConfig)
	if err != nil {
		return fmt.Errorf("failed to create fluent logger: %w", err)
	}

	return nil
}

func (o *DnstapFluentdOutput) write(frame []byte) error {
	dt := dnstap.Dnstap{}
	if err := proto.Unmarshal(frame, &dt); err != nil {
		return err
	}
	data, err := FlatDnstap(&dt, o.flatOption)
	if err != nil {
		return err
	}
	if err := o.client.Post(o.tag, *data); err != nil {
		return fmt.Errorf("failed to post fluent message, tag: %s %w", o.tag, err)
	}
	return nil
}

func (o *DnstapFluentdOutput) close() {
	o.client.Close()
}
