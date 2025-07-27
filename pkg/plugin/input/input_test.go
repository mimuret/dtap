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
	"context"
	"net"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"
	"github.com/mimuret/dtap/v3/pkg/buffer"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/plugin/input"
	"github.com/mimuret/dtap/v3/pkg/testtool"
	"github.com/mimuret/dtap/v3/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/net/nettest"
	"google.golang.org/protobuf/proto"
)

var _ types.InputPlugin = &dummyPlugin{}

type dummyPlugin struct {
	config.InputBlock
}

func (p *dummyPlugin) Start(ctx context.Context, forwarder types.Forwarder) error {
	return nil
}

type counter struct {
	I int
}

func (c *counter) Inc() {
	c.I++
}

var _ = Describe("InputServer", func() {
	Context("Serve", func() {
		var (
			srvErr    error
			ln        net.Listener
			srv       *input.InputServer
			buf       types.Writer
			forwarder *plugin.Forwarder
			dp        *dummyPlugin
		)
		BeforeEach(func() {
			dp = &dummyPlugin{
				InputBlock: config.InputBlock{
					Type: "dummy",
					Name: "default",
				},
			}
			srv = input.NewInputServer(dp, "", nil)
			srvErr = nil
			buf = buffer.NewRingBuffer(100, &counter{}, &counter{})
			forwarder = &plugin.Forwarder{}
			forwarder.SetupForwardTo([]types.Writer{buf})
			ln, srvErr = nettest.NewLocalListener("unix")
			Expect(srvErr).To(Succeed())
			go func() {
				srvErr = srv.Serve(context.Background(), forwarder, ln)
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
		AfterEach(func() {
			ln.Close()
		})
	})
	Context("Read", func() {
		var (
			srvErr    error
			connOut   net.Conn
			connIn    net.Conn
			srv       *input.InputServer
			buf       types.Buffer
			forwarder *plugin.Forwarder
		)
		BeforeEach(func() {
			srv = input.NewInputServer(&dummyPlugin{
				InputBlock: config.InputBlock{
					Type: "dummy",
					Name: "default",
				},
			}, input.FormatDNSTAP, nil)

			buf = buffer.NewRingBuffer(100, &counter{}, &counter{})
			forwarder = &plugin.Forwarder{}
			forwarder.SetupForwardTo([]types.Writer{buf})
			connOut, connIn = net.Pipe()
			srvErr = nil
			go func() {
				srvErr = srv.Read(context.Background(), forwarder, connOut)
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
