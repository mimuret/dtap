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
	"context"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"
	"github.com/fluent/fluent-logger-golang/fluent"
)

type FluetOutput interface {
	handle(*fluent.Fluent, *dnstap.Dnstap, chan error)
}
type Output interface {
	Run(context.Context, chan error)
	GetOutputChannel() chan []byte
	Finished() bool
}
type Input interface {
	Run(context.Context, chan []byte, chan error)
}

type SocketOutput interface {
	newConnect() (*framestream.Encoder, error)
}
