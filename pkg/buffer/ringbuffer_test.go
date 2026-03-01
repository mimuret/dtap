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
package buffer_test

import (
	"github.com/mimuret/dtap/v3/pkg/buffer"
	"github.com/mimuret/dtap/v3/pkg/testtool"
	"github.com/mimuret/dtap/v3/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type counter struct {
	I int
}

func (c *counter) Inc() {
	c.I++
}

var _ = Describe("RingBuffer", func() {
	var (
		rb     *buffer.RingBuffer
		c1, c2 *counter
		dm     *types.DnstapMessage
		dms    []*types.DnstapMessage
	)
	BeforeEach(func() {
		c1 = &counter{}
		c2 = &counter{}
		dms = nil
		rb = buffer.NewRingBuffer(10, c1, c2)
	})
	When("not lost", func() {
		BeforeEach(func() {
			for i := 0; i < 10; i++ {
				dm = testtool.CreateValidDnstapMessage()
				rb.Write(dm)
				dms = append(dms, dm)
			}
		})
		It("returns DnstapMessage", func() {
			recv := <-rb.Read()
			Expect(recv).To(Equal(dms[0]))
			Expect(c1.I).To(Equal(10))
			Expect(c2.I).To(Equal(0))
		})
	})
	When("lost", func() {
		BeforeEach(func() {
			for i := byte(0); i < byte(20); i++ {
				dm = testtool.CreateValidDnstapMessage()
				rb.Write(dm)
				dms = append(dms, dm)
			}
		})
		It("returns DnstapMessage", func() {
			recv := <-rb.Read()
			Expect(recv).To(Equal(dms[10]))
			Expect(c1.I).To(Equal(20))
			Expect(c2.I).To(Equal(10))
		})
	})
})
