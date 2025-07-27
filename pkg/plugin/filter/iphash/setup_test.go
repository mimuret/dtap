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
package iphash_test

import (
	"context"
	_ "embed"
	"net"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/mimuret/dtap/v3/pkg/plugin/filter/iphash"
	"github.com/mimuret/dtap/v3/pkg/testtool"
	"github.com/mimuret/dtap/v3/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("iphash", func() {
	Context("Setup", func() {
		var (
			err error
			fp  types.FilterPlugin
		)
		When("Salt not set", func() {
			BeforeEach(func() {
				fp, err = iphash.Setup(testtool.MustFilterBlock("filter", "filter_test", ``))
			})
			It("returns iphash", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp(`The argument "salt" is required`))
			})
		})
		When("Salt is an empty", func() {
			BeforeEach(func() {
				fp, err = iphash.Setup(testtool.MustFilterBlock("filter", "filter_test", `salt = ""`))
			})
			It("returns iphash", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("missing parameter Salt"))
			})
		})
		When("valid json", func() {
			BeforeEach(func() {
				fp, err = iphash.Setup(testtool.MustFilterBlock("filter", "filter_test", `salt = "hogehoge"`))
			})
			It("returns iphash", func() {
				Expect(err).To(Succeed())
				Expect(fp).NotTo(BeNil())
			})
		})
	})
	Context("Filter", func() {
		var (
			err error
			fp  types.FilterPlugin
			dm1 *types.DnstapMessage
			dm2 *types.DnstapMessage
			dt  *dnstap.Dnstap
		)
		BeforeEach(func() {
			dm1 = testtool.CreateValidDnstapMessage()
			dt = dm1.GetDnstap()
			fp, err = iphash.Setup(testtool.MustFilterBlock("filter", "filter_test", `salt = "hogehoge"`))
			Expect(err).To(Succeed())
		})
		When("protocol is IPv4", func() {
			BeforeEach(func() {
				dt.Message.SocketFamily = dnstap.SocketFamily_INET.Enum()
				dt.Message.ResponseAddress = []byte(net.IPv4(192, 168, 255, 255).To4())
				dt.Message.QueryAddress = []byte(net.IPv4(10, 0, 255, 255).To4())
				dm1, err = types.NewDnstapMessageFromDnstap(dt)
				Expect(err).To(Succeed())
				dm2 = fp.Filter(context.Background(), dm1)
			})
			It("adds hash", func() {
				Expect(dm2).NotTo(BeNil())
				Expect(dm2.Labels["QueryAddressHash"]).NotTo(BeEmpty())
				Expect(dm2.Labels["ResponseAddressHash"]).NotTo(BeEmpty())
			})
		})
		When("protocol is IPv6", func() {
			BeforeEach(func() {
				dt.Message.SocketFamily = dnstap.SocketFamily_INET6.Enum()
				dt.Message.ResponseAddress = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
				dt.Message.QueryAddress = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
				dm1, err = types.NewDnstapMessageFromDnstap(dt)
				Expect(err).To(Succeed())
				dm2 = fp.Filter(context.Background(), dm1)
			})
			It("adds hash", func() {
				Expect(dm2).NotTo(BeNil())
				Expect(dm2.Labels["QueryAddressHash"]).NotTo(BeEmpty())
				Expect(dm2.Labels["ResponseAddressHash"]).NotTo(BeEmpty())
			})
		})
	})
})
