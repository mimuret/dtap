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
package buffer

import (
	"sync"

	"github.com/mimuret/dtap/v2/pkg/types"
)

var _ types.Buffer = &RingBuffer{}

type dummyCounter struct{}

func (d *dummyCounter) Inc() {

}

type RingBuffer struct {
	sync.Mutex
	channel     chan *types.DnstapMessage
	inCounter   types.Counter
	lostCounter types.Counter
}

func NewRingBuffer(size uint, inCounter, lostCounter types.Counter) *RingBuffer {
	if inCounter == nil {
		inCounter = &dummyCounter{}
	}
	if lostCounter == nil {
		lostCounter = &dummyCounter{}
	}
	return &RingBuffer{
		channel:     make(chan *types.DnstapMessage, size),
		inCounter:   inCounter,
		lostCounter: lostCounter,
	}
}

func (r *RingBuffer) Read() <-chan *types.DnstapMessage {
	return r.channel
}

func (r *RingBuffer) Write(dt *types.DnstapMessage) {
	r.Lock()
	defer r.Unlock()
	select {
	case r.channel <- dt:
		r.inCounter.Inc()
	default:
		r.lostCounter.Inc()
		r.inCounter.Inc()
		<-r.channel
		r.channel <- dt
	}
}

func (r *RingBuffer) Close() {
	close(r.channel)
}
