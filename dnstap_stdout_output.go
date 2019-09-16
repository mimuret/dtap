/*
 * Copyright (c) 2019 Manabu Sonoda
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
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"
	"github.com/golang/protobuf/proto"
	"github.com/prometheus/common/log"
)

type DnstapStdoutOutput struct {
	config          *OutputStdoutConfig
	enc             *framestream.Encoder
	flatOption      DnstapFlatOption
	flushCancelFunc context.CancelFunc
}

func NewDnstapStdoutOutput(config *OutputStdoutConfig, params *DnstapOutputParams) *DnstapOutput {
	params.Handler = &DnstapStdoutOutput{
		config:     config,
		flatOption: &config.Flat,
	}
	return NewDnstapOutput(params)
}

func (o *DnstapStdoutOutput) open() error {
	return nil
}

func (o *DnstapStdoutOutput) write(frame []byte) error {
	dt := dnstap.Dnstap{}
	if err := proto.Unmarshal(frame, &dt); err != nil {
		return err
	}
	data, err := FlatDnstap(&dt, o.flatOption)
	if err != nil {
		return err
	}
	switch o.config.GetType() {
	case "json":
		buf, err := json.Marshal(data)
		if err != nil {
			return err
		}
		fmt.Println(string(buf))
	case "gotpl":
		buf := &bytes.Buffer{}
		if err := o.config.template.Execute(buf, data); err != nil {
			return err
		}
		fmt.Println(buf.String())
	default:
		log.Fatalf("unsupported Type %s", o.config.GetType())
	}
	return nil
}

func (o *DnstapStdoutOutput) close() {
}
