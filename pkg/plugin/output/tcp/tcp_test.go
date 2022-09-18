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
	"io"
	"net"

	"github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/plugin/output/tcp"
	"github.com/mimuret/dtap/v2/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("output/tcp", func() {
	Context("Setup", func() {
		var (
			op  types.OutputPlugin
			err error
		)
		When("type mismatch", func() {
			BeforeEach(func() {
				op, err = tcp.Setup(json.RawMessage(`{"Name": "tcp", "Host": 0}`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to decode config"))
			})
		})
		When("Host is an empty", func() {
			BeforeEach(func() {
				op, err = tcp.Setup(json.RawMessage(`{"Name": "tcp"}`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("missing parameter Host"))
			})
		})
		When("Port is an empty", func() {
			BeforeEach(func() {
				op, err = tcp.Setup(json.RawMessage(`{"Name": "tcp", "Host": "127.0.0.1"}`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("missing parameter Port"))
			})
		})
		When("valid config", func() {
			BeforeEach(func() {
				op, err = tcp.Setup(json.RawMessage(`{"Name": "tcp", "Host": "127.0.0.1","Port": 10053}`))
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
			ln, err = net.Listen("tcp", "127.0.0.1:10053")
			Expect(err).To(Succeed())
			op, err = tcp.Setup(json.RawMessage(`{"Name": "tcp", "Host": "127.0.0.1","Port": 10053}`))
			Expect(err).To(Succeed())
			p = op.(*tcp.TCP)
		})
		AfterEach(func() {
			ln.Close()
		})
		When("failed to connect", func() {
			BeforeEach(func() {
				p.Port = 20053
				conn, err = p.NewConnect()
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to connect tcp socket"))
			})
		})
		When("valid", func() {
			BeforeEach(func() {
				conn, err = p.NewConnect()
			})
			It("Succeed", func() {
				Expect(err).To(Succeed())
				Expect(conn).NotTo(BeNil())
			})
		})
	})
})
