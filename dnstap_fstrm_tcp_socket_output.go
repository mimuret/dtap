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
	"net"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"
)

type DnstapFstrmTCPSocketOutput struct {
	config *OutputTCPSocketConfig
}

func NewDnstapFstrmTCPSocketOutput(config *OutputTCPSocketConfig, params *DnstapOutputParams) *DnstapOutput {
	tcp := &DnstapFstrmTCPSocketOutput{config: config}
	return NewDnstapFstrmSocketOutput(tcp, params)
}

func (o *DnstapFstrmTCPSocketOutput) newConnect() (*framestream.Encoder, error) {
	w, err := net.Dial("tcp", o.config.GetAddress())
	if err != nil {

		return nil, fmt.Errorf("failed to connect tcp socket, address: %s err: %w", o.config.GetAddress(), err)
	}
	enc, err := framestream.NewEncoder(w, &framestream.EncoderOptions{ContentType: dnstap.FSContentType, Bidirectional: true})
	if err != nil {
		return nil, fmt.Errorf("failed to create fstrm encorder, address: %s err: %w", o.config.GetAddress(), err)
	}
	return enc, nil
}
