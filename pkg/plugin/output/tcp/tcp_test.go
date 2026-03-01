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
	"io"
	"net"

	"golang.org/x/net/nettest"

	"github.com/mimuret/dtap/v3/pkg/plugin/output/tcp"
	"github.com/mimuret/dtap/v3/pkg/testtool"
	"github.com/mimuret/dtap/v3/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("output/tcp", func() {
	Context("Setup", func() {
		var (
			op  types.OutputPlugin
			err error
		)
		BeforeEach(func() {
		})
		When("Host is an empty", func() {
			BeforeEach(func() {
				op, err = tcp.Setup(testtool.MustOutputBlock("tcp", "tcp_test", ``))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("missing parameter Host"))
			})
		})
		When("Port is an empty", func() {
			BeforeEach(func() {
				op, err = tcp.Setup(testtool.MustOutputBlock("tcp", "tcp_test", `host = "127.0.0.1" `))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("missing parameter Port"))
			})
		})
		When("valid config", func() {
			BeforeEach(func() {
				op, err = tcp.Setup(testtool.MustOutputBlock("tcp", "tcp_test", `
				host = "127.0.0.1"
				port = 10053
				`))
			})
			It("returns error", func() {
				Expect(err).To(Succeed())
				Expect(op).NotTo(BeNil())
			})
		})
	})
	Context("NewConnect", func() {
		var (
			op   types.OutputPlugin
			p    *tcp.TCP
			err  error
			ln   net.Listener
			conn io.Writer
		)
		BeforeEach(func() {
			ln, err = nettest.NewLocalListener("tcp")
			Expect(err).To(Succeed())
			addr, ok := ln.Addr().(*net.TCPAddr)
			Expect(ok).To(BeTrue())
			op, err = tcp.Setup(testtool.MustOutputBlock("tcp", "tcp_test", `
			host = "127.0.0.1"
			port = 10053
			`))
			Expect(err).To(Succeed())
			p = op.(*tcp.TCP)
			p.Port = uint16(addr.Port)
		})
		AfterEach(func() {
			ln.Close()
		})
		When("failed to connect", func() {
			BeforeEach(func() {
				p.Port = 20053
				conn, err = p.NewConnect(context.Background())
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to connect tcp socket"))
			})
		})
		When("valid", func() {
			BeforeEach(func() {
				conn, err = p.NewConnect(context.Background())
			})
			It("Succeed", func() {
				Expect(err).To(Succeed())
				Expect(conn).NotTo(BeNil())
			})
		})
	})
})
