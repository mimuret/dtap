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
	"net"

	"github.com/goccy/go-json"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/nettest"

	"github.com/mimuret/dtap/v2/pkg/plugin/input/tcp"
	"github.com/mimuret/dtap/v2/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("input/tcp", func() {
	Context("SetupTCPSocket", func() {
		var (
			err error
			p   types.InputPlugin
		)
		BeforeEach(func() {
			prometheus.DefaultRegisterer = prometheus.NewRegistry()
		})
		When("Type missmatch", func() {
			BeforeEach(func() {
				p, err = tcp.SetupTCPSocket(json.RawMessage(`{"Name": "tcp","ID":"id1","Address": 0}`))
			})
			It("returns error", func() {
				Expect(p).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to decode config"))
			})
		})
		When("Port is an empty", func() {
			BeforeEach(func() {
				p, err = tcp.SetupTCPSocket(json.RawMessage(`{"Name":"tcp","ID":"id2","Address":"0.0.0.0"}`))
			})
			It("returns error", func() {
				Expect(p).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("missing parameter Port"))
			})
		})
		When("valid config", func() {
			BeforeEach(func() {
				p, err = tcp.SetupTCPSocket(json.RawMessage(`{"Name":"tcp","ID":"id3","Port": 10053}`))
			})
			It("returns error", func() {
				Expect(err).To(Succeed())
				Expect(p.(*tcp.TCPSocket).Address).To(Equal(""))
				Expect(p.(*tcp.TCPSocket).Port).To(Equal(uint16(10053)))
			})
		})
	})
	Context("Listen", func() {
		var (
			err error
			ip  types.InputPlugin
			p   *tcp.TCPSocket
		)
		BeforeEach(func() {
			ln, lerr := nettest.NewLocalListener("tcp")
			Expect(lerr).To(Succeed())
			addr, ok := ln.Addr().(*net.TCPAddr)
			Expect(ok).To(BeTrue())
			ln.Close()

			prometheus.DefaultRegisterer = prometheus.NewRegistry()
			ip, err = tcp.SetupTCPSocket(json.RawMessage(`{"Name":"tcp","ID":"id4","Port": 10053}`))
			Expect(err).To(Succeed())
			p = ip.(*tcp.TCPSocket)
			p.Port = uint16(addr.Port)
		})
		When("failed to listen", func() {
			BeforeEach(func() {
				p.Address = "example.jp"
				err = p.Listen()
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to listen"))
			})
		})
		When("open socket", func() {
			BeforeEach(func() {
				err = p.Listen()
			})
			AfterEach(func() {
				p.Close()
			})
			It("succeed", func() {
				Expect(err).To(Succeed())
			})
		})
	})
})
