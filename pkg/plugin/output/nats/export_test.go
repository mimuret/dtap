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
package nats

import (
	"github.com/mimuret/dtap/v2/pkg/plugin/pub"
	"github.com/nats-io/nats.go"
)

func GetStatics(p *Nats) nats.Statistics {
	return p.conn.Statistics
}
func GetPublisher(p *Nats) pub.Publisher {
	return p.publisher
}
