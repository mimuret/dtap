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
package input_test

import (
	"net"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"
	"github.com/mimuret/dtap/v2/pkg/buffer"
	"github.com/mimuret/dtap/v2/pkg/plugin/input"
	"github.com/mimuret/dtap/v2/pkg/testtool"
	"github.com/mimuret/dtap/v2/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/nettest"
	"google.golang.org/protobuf/proto"
)

type counter struct {
	I int
}

func (c *counter) Inc() {
	c.I++
}

var _ = Describe("InputServer", func() {
	Context("Serve", func() {
		var (
			srvErr error
			ln     net.Listener
			srv    *input.InputServer
			buf    types.Writer
		)
		BeforeEach(func() {
			srv = input.NewInputServer(nil)
			srvErr = nil
			buf = buffer.NewRingBuffer(100, &counter{}, &counter{})
			ln, srvErr = nettest.NewLocalListener("unix")
			Expect(srvErr).To(Succeed())
			go func() {
				srvErr = srv.Serve(ln, buf)
			}()
		})
		When("write message", func() {
			BeforeEach(func() {
				conn, cerr := net.DialUnix(ln.Addr().Network(), nil, ln.Addr().(*net.UnixAddr))
				Expect(cerr).To(Succeed())
				_, cerr = conn.Write([]byte("hogehoge"))
				Expect(cerr).To(Succeed())
			})
			It("accept connection", func() {
				Expect(srvErr).To(Succeed())
			})
		})
		When("close listener", func() {
			BeforeEach(func() {
				ln.Close()
			})
			It("returns ErrClosed", func() {
				Eventually(func() error { return srvErr }).Should(MatchError(net.ErrClosed))
			})
		})
		AfterEach(func() {
			ln.Close()
		})
	})
	Context("Read", func() {
		var (
			srvErr  error
			connOut net.Conn
			connIn  net.Conn
			srv     *input.InputServer
			buf     types.Buffer
		)
		BeforeEach(func() {
			srv = input.NewInputServer(nil)
			buf = buffer.NewRingBuffer(100, &counter{}, &counter{})
			connOut, connIn = net.Pipe()
			srvErr = nil
			go func() {
				srvErr = srv.Read(connOut, buf)
			}()
		})
		AfterEach(func() {
			connIn.Close()
			connOut.Close()
		})
		When("invalid message", func() {
			BeforeEach(func() {
				_, cerr := connIn.Write([]byte("hogehoge"))
				Expect(cerr).To(Succeed())
			})
			It("returns decode error", func() {
				Eventually(func() error { return srvErr }).Should(MatchError(framestream.ErrDecode))
			})
		})
		When("valid message", func() {
			var (
				m *types.DnstapMessage
			)
			BeforeEach(func() {
				enc, cerr := framestream.NewEncoder(connIn, &framestream.EncoderOptions{
					ContentType:   dnstap.FSContentType,
					Bidirectional: true,
				})
				Expect(cerr).To(Succeed())

				msg := testtool.CreateValidDnstapMessage()
				bs, cerr := proto.Marshal(msg.GetDnstap())
				Expect(cerr).To(Succeed())

				_, cerr = enc.Write(bs)
				Expect(cerr).To(Succeed())

				enc.Flush()
			})
			It("returns decode error", func() {
				Eventually(func() error { return srvErr }).Should(Succeed())
				m = <-buf.Read()
				Expect(m).NotTo(BeNil())
			})
		})
	})
})
