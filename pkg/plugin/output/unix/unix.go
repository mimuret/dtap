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
package unix

import (
	"io"
	"net"
	"time"

	"github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/output"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
)

func init() {
	_ = registry.RegisterOutputPlugin("unix", Setup)
}

func Setup(bs json.RawMessage) (types.OutputPlugin, error) {
	s := &Unix{}
	if err := json.Unmarshal(bs, s); err != nil {
		return nil, errors.Wrap(err, "failed to decode config")
	}
	if s.Path == "" {
		return nil, errors.New("missing parameter Path")
	}
	s.DnstapOutput = output.NewDnstapOutput(output.NewDnstapFstrmSocketOutput(s, time.Second, nil))
	return s, nil
}

var _ types.OutputPlugin = &Unix{}
var _ output.SocketOutput = &Unix{}

type Unix struct {
	plugin.PluginCommon
	*output.DnstapOutput

	Path string

	w net.Conn
}

func (f *Unix) NewConnect() (io.Writer, error) {
	var err error
	f.w, err = net.Dial("unix", f.Path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect unix socket, path: %s", f.Path)
	}
	return f.w, nil
}

func (f *Unix) Close() {
	f.w.Close()
}
