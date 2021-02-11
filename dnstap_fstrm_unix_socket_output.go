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

type DnstapFstrmUnixSockOutput struct {
	config *OutputUnixSocketConfig
}

func NewDnstapFstrmUnixSockOutput(config *OutputUnixSocketConfig, params *DnstapOutputParams) *DnstapOutput {
	unix := &DnstapFstrmUnixSockOutput{
		config: config,
	}
	return NewDnstapFstrmSocketOutput(unix, params)
}

func (o *DnstapFstrmUnixSockOutput) newConnect() (*framestream.Encoder, error) {
	w, err := net.Dial("unix", o.config.GetPath())
	if err != nil {
		return nil, fmt.Errorf("failed to connect unix socket, path: %s: %w", o.config.GetPath(), err)
	}
	enc, err := framestream.NewEncoder(w, &framestream.EncoderOptions{ContentType: dnstap.FSContentType, Bidirectional: true})
	if err != nil {
		return nil, fmt.Errorf("failed to create fstrm encorder, path: %s: %w", o.config.GetPath(), err)
	}
	return enc, nil
}
