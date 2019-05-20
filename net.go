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
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Net struct {
	IP           net.IP
	PrefixLength int
}

func (n *Net) String() string {
	return n.IP.String() + "/" + strconv.Itoa(n.PrefixLength)
}

func (n Net) MarshalJSON() ([]byte, error) {
	if n.IP == nil {
		return []byte(`"<nil>"`), nil
	}
	return []byte(`"` + n.String() + `"`), nil
}

func (n *Net) UnmarshalJSON(b []byte) error {
	var err error
	str := string(b)
	if str == `"<nil>"` {
		return nil
	}
	strlen := len(str)
	if strlen < 3 || str[0] != '"' || str[strlen-1] != '"' {
		return fmt.Errorf("can't parse Net: %v", str)
	}
	bs := strings.Split(string(b[1:strlen-2]), "/")

	n.IP = net.ParseIP(bs[0])
	if n.IP == nil {
		return fmt.Errorf("can't parse Net: %v", bs[0])
	}
	if len(bs) > 1 {
		n.PrefixLength, err = strconv.Atoi(bs[1])
		if err != nil {
			return fmt.Errorf("can't parse Prefixlength: %v", bs[1])
		}
	}
	return nil
}
