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
package tcp

import (
	"io"
	"net"
	"strconv"
	"time"

	"github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/output"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
)

func init() {
	_ = registry.RegisterOutputPlugin("tcp", Setup)
}

func Setup(bs json.RawMessage) (types.OutputPlugin, error) {
	s := &TCP{}
	if err := json.Unmarshal(bs, s); err != nil {
		return nil, errors.Wrap(err, "failed to decode config")
	}
	if s.Host == "" {
		return nil, errors.Errorf("missing parameter Host")
	}
	if s.Port == 0 {
		return nil, errors.Errorf("missing parameter Port")
	}
	s.DnstapOutput = output.NewDnstapOutput(output.NewDnstapFstrmSocketOutput(s, time.Second, nil))
	return s, nil
}

var _ types.OutputPlugin = &TCP{}
var _ output.SocketOutput = &TCP{}

type TCP struct {
	plugin.PluginCommon
	*output.DnstapOutput
	Host string
	Port uint16

	w net.Conn
}

func (t *TCP) NewConnect() (io.Writer, error) {
	var err error
	target := net.JoinHostPort(t.Host, strconv.Itoa(int(t.Port)))
	t.w, err = net.Dial("tcp", target)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect tcp socket, address: %s", target)
	}
	return t.w, nil
}

func (t *TCP) Close() {
	t.w.Close()
}
