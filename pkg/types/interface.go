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

package types

import (
	"context"
)

type Buffer interface {
	Writer
	Reader
}

type Writer interface {
	Write(*DnstapMessage)
}

type Reader interface {
	Read() <-chan *DnstapMessage
}

type Counter interface {
	Inc()
}

// plugin interfaes

type Plugin interface {
	GetFullName() string
	GetName() string
	GetType() string
}

type Receiver interface {
	Writer
}

type Forwarder interface {
	SetupForwardTo([]Writer)
	Forward(*DnstapMessage)
}

type InputPlugin interface {
	Plugin
	Start(context.Context, Forwarder) error
}

type FilterPlugin interface {
	Plugin
	Filter(context.Context, *DnstapMessage) *DnstapMessage
}

type OutputPlugin interface {
	Plugin
	Start(context.Context, Reader) error
	// 同時実行可能数を取得する
	MaxConcurrent() uint
}
