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
package output_test

import (
	"context"
	"errors"
	"sync"

	"github.com/mimuret/dtap/v3/pkg/buffer"
	"github.com/mimuret/dtap/v3/pkg/plugin/output"
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

type handler struct {
	ErrWrite error
	ErrOpen  error
	Opened   bool
	Closed   bool
	Msg      *types.DnstapMessage
}

func (h *handler) Open(context.Context) error {
	h.Opened = true
	return h.ErrOpen
}
func (h *handler) Write(ctx context.Context, msg *types.DnstapMessage) error {
	h.Msg = msg
	return h.ErrWrite
}
func (h *handler) Close(ctx context.Context) {
	h.Closed = true
}

var _ = Describe("DnstapOutput", func() {
	Context("Serve", func() {
		var (
			h          *handler
			out        *output.DnstapOutput
			msg        *types.DnstapMessage
			ctx        context.Context
			cancelFunc context.CancelFunc
			wg         *sync.WaitGroup
			buf        types.Buffer
		)
		BeforeEach(func() {
			buf = buffer.NewRingBuffer(100, &counter{}, &counter{})
			ctx, cancelFunc = context.WithCancel(context.Background())
			wg = &sync.WaitGroup{}
			h = &handler{}
			out = output.NewDnstapOutput(h, 0)
			msg = testtool.CreateValidDnstapMessage()
		})
		Context("incoming message", func() {
			BeforeEach(func() {
				buf.Write(msg)
				wg.Add(1)
				go func() {
					err := out.Start(ctx, buf)
					Expect(err).To(Succeed())
					wg.Done()
				}()
			})
			AfterEach(func() {
				cancelFunc()
				wg.Wait()
			})
			It("write handle", func() {
				Eventually(func() bool { return h.Opened }).Should(BeTrue())
				Eventually(func() bool { return h.Closed }).Should(BeFalse())
				Eventually(h.Msg).ShouldNot(BeNil())
				Expect(h.Msg).To(Equal(msg))
			})
		})
		Context("close", func() {
			BeforeEach(func() {
				wg.Add(1)
				go func() {
					err := out.Start(ctx, buf)
					Expect(err).To(Succeed())
					wg.Done()
				}()
				Eventually(func() bool { return h.Opened }).Should(BeTrue())
				cancelFunc()
				wg.Wait()
			})
			It("run Close", func() {
				Eventually(func() bool { return h.Closed }).Should(BeTrue())
			})
		})
		Context("retry open", func() {
			BeforeEach(func() {
				h.ErrOpen = errors.New("dummy")
				wg.Add(1)
				go func() {
					err := out.Start(ctx, buf)
					Expect(err).To(Succeed())
					wg.Done()
				}()
				Eventually(func() bool { return h.Opened }).Should(BeTrue())
				h.ErrOpen = nil
				h.Opened = false
				buf.Write(msg)
			})
			It("write handle", func() {
				Eventually(func() bool { return h.Opened }, "10s").Should(BeTrue())
				Eventually(h.Msg).ShouldNot(BeNil())
				Expect(h.Msg).To(Equal(msg))
			})
			AfterEach(func() {
				cancelFunc()
				wg.Wait()
			})
		})
	})
})
