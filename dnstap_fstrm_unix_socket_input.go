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
	"os"
	"os/user"
	"strconv"
)

func NewDnstapFstrmUnixSocketInput(config *InputUnixSocketConfig) (*DnstapFstrmSocketInput, error) {
	os.Remove(config.GetPath())
	l, err := net.Listen("unix", config.GetPath())
	if err != nil {
		return nil, fmt.Errorf("failed to listen %s: %w", config.GetPath(), err)
	}
	if config.GetUser() != "" {
		if u, err := user.Lookup(config.GetUser()); err != nil {
			return nil, fmt.Errorf("failed to get chown user %s: %w", config.GetUser(), err)
		} else {
			uid, err := strconv.Atoi(u.Uid)
			if err != nil {
				return nil, fmt.Errorf("failed to get uid: %w", err)
			}
			gid, err := strconv.Atoi(u.Gid)
			if err != nil {
				return nil, fmt.Errorf("failed to get gid: %w", err)
			}
			if err := os.Chown(config.GetPath(), uid, gid); err != nil {
				return nil, fmt.Errorf("failed to change owner %s (%s:%s): %w", config.GetUser(), u.Uid, u.Gid, err)
			}
		}
	}
	return NewDnstapFstrmSocketInput(l)
}
