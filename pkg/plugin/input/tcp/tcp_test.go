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
package tcp_test

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/plugin/input/tcp"
	"github.com/mimuret/dtap/v3/pkg/testtool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TCPSocket", func() {
	var (
		ctx       context.Context
		cancel    context.CancelFunc
		forwarder *plugin.Forwarder
		tcpSocket *tcp.TCPSocket
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		forwarder = &plugin.Forwarder{}
	})

	AfterEach(func() {
		cancel()
	})

	Context("Setup", func() {
		It("should successfully setup with valid HCL", func() {
			plugin, err := tcp.Setup(testtool.MustInputBlock("tcp", "test_tcp", `
address = "127.0.0.1"
port = 12345
`))
			Expect(err).To(Succeed())
			Expect(plugin).ToNot(BeNil())

			tcpSocket = plugin.(*tcp.TCPSocket)
			Expect(tcpSocket.Address).To(Equal("127.0.0.1"))
			Expect(tcpSocket.Port).To(Equal(uint16(12345)))
			Expect(tcpSocket.Format).To(Equal("DNSTAP"))
		})

		It("should return an error if Port is missing", func() {
			plugin, err := tcp.Setup(testtool.MustInputBlock("tcp", "test_tcp", `
address = "127.0.0.1"
`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`Missing required argument; The argument "port" is required`))
			Expect(plugin).To(BeNil())
		})
	})

	Context("Start", func() {
		It("should start and stop the server successfully", func() {
			p, err := tcp.Setup(testtool.MustInputBlock("tcp", "test_tcp", `
address = "127.0.0.1"
port = 12345
format = "DNSTAP"
`))
			Expect(err).To(Succeed())

			tcpSocket := p.(*tcp.TCPSocket)
			go func() {
				err := tcpSocket.Start(ctx, forwarder)
				Expect(err).To(Succeed())
			}()

			// Wait for the server to start
			time.Sleep(1 * time.Second)

			// Connect to the server
			conn, err := net.Dial("tcp", net.JoinHostPort(tcpSocket.Address, strconv.Itoa(int(tcpSocket.Port))))
			Expect(err).To(Succeed())
			Expect(conn).ToNot(BeNil())
			conn.Close()

			// Stop the server
			cancel()
			time.Sleep(1 * time.Second) // Allow time for the server to shut down
		})
	})
})
