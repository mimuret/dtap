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
	"net"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"
	"github.com/pkg/errors"
)

type DnstapFstrmTCPSocketOutput struct {
	config *OutputTCPSocketConfig
}

func NewDnstapFstrmTCPSocketOutput(config *OutputTCPSocketConfig) *DnstapOutput {
	tcp := &DnstapFstrmTCPSocketOutput{config: config}
	return NewDnstapFstrmSocketOutput(config.Buffer.GetBufferSize(), tcp)
}

func (o *DnstapFstrmTCPSocketOutput) newConnect() (*framestream.Encoder, error) {
	w, err := net.Dial("tcp", o.config.GetAddress())
	if err != nil {

		return nil, errors.Wrapf(err, "can't connect tcp socket, address: %s", o.config.GetAddress())
	}
	enc, err := framestream.NewEncoder(w, &framestream.EncoderOptions{ContentType: dnstap.FSContentType, Bidirectional: true})
	if err != nil {

		return nil, errors.Wrapf(err, "can't create fstrm encorder, address: %s", o.config.GetAddress())
	}
	return enc, nil
}
