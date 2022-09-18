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
package mask_test

import (
	_ "embed"
	"net"

	"github.com/goccy/go-json"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/mimuret/dtap/v2/pkg/plugin/filter/mask"
	"github.com/mimuret/dtap/v2/pkg/testtool"
	"github.com/mimuret/dtap/v2/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("mask", func() {
	Context("Setup", func() {
		var (
			err error
			fp  types.FilterPlugin
		)
		When("invalid json", func() {
			BeforeEach(func() {
				fp, err = mask.Setup(json.RawMessage(`{"Name": 0}`))
			})
			It("returns mask", func() {
				Expect(err).To(HaveOccurred())
			})
		})
		When("MaskLen4 is invalid", func() {
			BeforeEach(func() {
				fp, err = mask.Setup(json.RawMessage(`{"Name": "mask", "MaskLen4": 33}`))
			})
			It("returns mask", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("invalid value MaskLen4"))
			})
		})
		When("MaskLen6 is invalid", func() {
			BeforeEach(func() {
				fp, err = mask.Setup(json.RawMessage(`{"Name": "mask", "MaskLen6": 129}`))
			})
			It("returns mask", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("invalid value MaskLen6"))
			})
		})
		When("valid json", func() {
			BeforeEach(func() {
				fp, err = mask.Setup(json.RawMessage(`{"Name": "mask"}`))
			})
			It("returns mask", func() {
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
		})
		When("protocol is IPv4", func() {
			BeforeEach(func() {
				dt.Message.SocketFamily = dnstap.SocketFamily_INET.Enum()
				dt.Message.ResponseAddress = []byte(net.IPv4(192, 168, 255, 255).To4())
				dt.Message.QueryAddress = []byte(net.IPv4(10, 0, 255, 255).To4())
				dm1, err = types.NewDnstapMessageFromDnstap(dt)
				Expect(err).To(Succeed())
			})
			When("MaskLen4 is 32", func() {
				BeforeEach(func() {
					fp, err = mask.Setup(json.RawMessage(`{"Name": "mask","MaskLen4":32}`))
					Expect(err).To(Succeed())
					dm2 = fp.Filter(dm1)
				})
				It("mask /32", func() {
					Expect(dm2).NotTo(BeNil())
					Expect(dm2.GetDnstap().GetMessage().ResponseAddress).To(Equal([]byte{192, 168, 255, 255}))
					Expect(dm2.GetDnstap().GetMessage().QueryAddress).To(Equal([]byte{10, 0, 255, 255}))
				})
			})
			When("MaskLen4 default", func() {
				BeforeEach(func() {
					fp, err = mask.Setup(json.RawMessage(`{"Name": "mask"}`))
					Expect(err).To(Succeed())
					dm2 = fp.Filter(dm1)
				})
				It("mask /22", func() {
					Expect(dm2).NotTo(BeNil())
					Expect(dm2.GetDnstap().GetMessage().ResponseAddress).To(Equal([]byte{192, 168, 252, 0}))
					Expect(dm2.GetDnstap().GetMessage().QueryAddress).To(Equal([]byte{10, 0, 252, 0}))
				})
			})
			When("MaskLen4 is 0", func() {
				BeforeEach(func() {
					fp, err = mask.Setup(json.RawMessage(`{"Name": "mask","MaskLen4":0}`))
					Expect(err).To(Succeed())
					dm2 = fp.Filter(dm1)
				})
				It("mask /0", func() {
					Expect(dm2).NotTo(BeNil())
					Expect(dm2.GetDnstap().GetMessage().ResponseAddress).To(Equal([]byte{0, 0, 0, 0}))
					Expect(dm2.GetDnstap().GetMessage().QueryAddress).To(Equal([]byte{0, 0, 0, 0}))
				})
			})
		})
		When("protocol is IPv6", func() {
			BeforeEach(func() {
				dt.Message.SocketFamily = dnstap.SocketFamily_INET6.Enum()
				dt.Message.ResponseAddress = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
				dt.Message.QueryAddress = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
				dm1, err = types.NewDnstapMessageFromDnstap(dt)
				Expect(err).To(Succeed())
			})
			When("MaskLen6 is 128", func() {
				BeforeEach(func() {
					fp, err = mask.Setup(json.RawMessage(`{"Name": "mask","MaskLen6":128}`))
					Expect(err).To(Succeed())
					dm2 = fp.Filter(dm1)
				})
				It("mask /128", func() {
					Expect(dm2).NotTo(BeNil())
					Expect(dm2.GetDnstap().GetMessage().ResponseAddress).To(Equal([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}))
					Expect(dm2.GetDnstap().GetMessage().QueryAddress).To(Equal([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}))
				})
			})
			When("MaskLen6 default", func() {
				BeforeEach(func() {
					fp, err = mask.Setup(json.RawMessage(`{"Name": "mask"}`))
					Expect(err).To(Succeed())
					dm2 = fp.Filter(dm1)
				})
				It("mask /40", func() {
					Expect(dm2).NotTo(BeNil())
					Expect(dm2.GetDnstap().GetMessage().ResponseAddress).To(Equal([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}))
					Expect(dm2.GetDnstap().GetMessage().QueryAddress).To(Equal([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}))
				})
			})
			When("MaskLen6 is 0", func() {
				BeforeEach(func() {
					fp, err = mask.Setup(json.RawMessage(`{"Name": "mask","MaskLen6":0}`))
					Expect(err).To(Succeed())
					dm2 = fp.Filter(dm1)
				})
				It("mask /0", func() {
					Expect(dm2).NotTo(BeNil())
					Expect(dm2.GetDnstap().GetMessage().ResponseAddress).To(Equal([]byte(net.IPv6zero)))
					Expect(dm2.GetDnstap().GetMessage().QueryAddress).To(Equal([]byte(net.IPv6zero)))
				})
			})
		})
	})
})
