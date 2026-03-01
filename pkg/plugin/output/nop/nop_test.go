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
package nop_test

import (
	"context"

	"github.com/mimuret/dtap/v3/pkg/buffer"
	"github.com/mimuret/dtap/v3/pkg/plugin/output/nop"
	"github.com/mimuret/dtap/v3/pkg/testtool"
	"github.com/mimuret/dtap/v3/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("output/nop", func() {
	Context("Setup", func() {
		var (
			op  types.OutputPlugin
			err error
		)
		BeforeEach(func() {
			op, err = nop.Setup(testtool.MustOutputBlock("nop", "nop_test", ``))
		})
		It("returns error", func() {
			Expect(err).To(Succeed())
			Expect(op).NotTo(BeNil())
		})
	})
	Context("Start", func() {
		var (
			op         types.OutputPlugin
			err        error
			buf        types.Buffer
			ctx        context.Context
			cancelFunc context.CancelFunc
		)
		BeforeEach(func() {
			buf = buffer.NewRingBuffer(100, nil, nil)
			for i := 0; i < 100; i++ {
				buf.Write(&types.DnstapMessage{})
			}
			Expect(len(buf.Read())).To(Equal(100))
			op, err = nop.Setup(testtool.MustOutputBlock("nop", "nop_test", ``))
			Expect(err).To(Succeed())
			ctx, cancelFunc = context.WithCancel(context.Background())

			go func() {
				err := op.Start(ctx, buf)
				Expect(err).To(Succeed())
			}()
		})
		AfterEach(func() {
			cancelFunc()
		})
		It("returns error", func() {
			Eventually(func() int {
				return len(buf.Read())
			}).Should(Equal(0))
		})
	})
})
