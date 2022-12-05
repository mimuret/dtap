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
	"fmt"
	"io"
	"net"
	"time"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/mimuret/dtap/v2/pkg/plugin/output"
	"github.com/mimuret/dtap/v2/pkg/testtool"
	"github.com/mimuret/dtap/v2/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type socketOutput struct {
	ErrNewConnect error
	RunNewConnect int
	RunClose      int
	Conn          net.Conn
}

func (s *socketOutput) NewConnect() (io.Writer, error) {
	s.RunNewConnect++
	if s.ErrNewConnect != nil {
		return nil, s.ErrNewConnect
	}
	return net.Dial("tcp", "127.0.0.1:10153")
}

func (s *socketOutput) Close() {
	s.RunClose++
}

var _ = Describe("DnstapFstrmSocketOutput", func() {
	Context("Serve", func() {
		var (
			err      error
			so       *socketOutput
			h        *output.DnstapFstrmSocketOutput
			msg      *types.DnstapMessage
			server   net.Listener
			resQueue chan []byte
			iServer  *dnstap.FrameStreamSockInput
		)
		BeforeEach(func() {
			resQueue = make(chan []byte, 100)
			server, err = net.Listen("tcp", "127.0.0.1:10153")
			Expect(err).To(Succeed())
			iServer = dnstap.NewFrameStreamSockInput(server)

			Expect(err).To(Succeed())

			so = &socketOutput{}
			h = output.NewDnstapFstrmSocketOutput(so, time.Second, nil)
			msg = testtool.CreateValidDnstapMessage()
		})
		AfterEach(func() {
			server.Close()
		})
		Context("Open", func() {
			When("failed to dial", func() {
				BeforeEach(func() {
					so.ErrNewConnect = fmt.Errorf("dummy")
					err = h.Open()
				})
				It("returns error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("failed to connect socket"))
					Expect(so.RunClose).To(Equal(0))
				})
			})
			When("failed to negotiate", func() {
				BeforeEach(func() {
					err = h.Open()
				})
				It("returns error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("failed to create fstrm encoder"))
					Expect(so.RunClose).To(Equal(1))
				})
			})
			When("create encoder", func() {
				BeforeEach(func() {
					go func() {
						iServer.ReadInto(resQueue)
					}()
					err = h.Open()
				})
				AfterEach(func() {
					h.Close()
				})
				It("returns error", func() {
					Expect(err).To(Succeed())
					Expect(so.RunNewConnect).To(Equal(1))
					Expect(so.RunClose).To(Equal(0))
				})
			})
		})
		Context("Write", func() {
			BeforeEach(func() {
				go func() {
					iServer.ReadInto(resQueue)
				}()
				err := h.Open()
				Expect(err).To(Succeed())

				defer h.Close()
				Expect(so.RunNewConnect).To(Equal(1))
				err = h.Write(msg)
				Expect(err).To(Succeed())
			})
			It("write handle", func() {
				m := <-resQueue
				Expect(m).To(Equal(msg.GetRaw()))
			})
		})
	})
})
