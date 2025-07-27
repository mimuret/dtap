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
	"context"
	"net"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	"github.com/mimuret/dtap/v3/pkg/plugin/filter/mask"
	"github.com/mimuret/dtap/v3/pkg/testtool"
	"github.com/mimuret/dtap/v3/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Mask", func() {
	var (
		ctx         context.Context
		dnsMsg      *dns.Msg
		dnsMsgBytes []byte
	)

	BeforeEach(func() {
		ctx = context.Background()
		dnsMsg = &dns.Msg{}
		dnsMsg.SetQuestion("example.com.", dns.TypeA)
		dnsMsgBytes, _ = dnsMsg.Pack()
	})

	Context("Setup", func() {
		It("should successfully setup with valid HCL", func() {
			hclData := `
mask_len4 = 24
mask_len6 = 64
query_address_enabled = true
response_address_enabled = false
`
			plugin, err := mask.Setup(testtool.MustFilterBlock("mask", "test_mask", hclData))
			Expect(err).To(Succeed())
			Expect(plugin).ToNot(BeNil())
		})

		It("should return an error for invalid MaskLen4", func() {
			hclData := `
mask_len4 = 33
`
			plugin, err := mask.Setup(testtool.MustFilterBlock("mask", "test_mask", hclData))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("MaskLen4 must be between 0 and 32"))
			Expect(plugin).To(BeNil())
		})

		It("should return an error for invalid MaskLen6", func() {
			hclData := `
mask_len6 = 129
`

			plugin, err := mask.Setup(testtool.MustFilterBlock("mask", "test_mask", hclData))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("MaskLen6 must be between 0 and 128"))
			Expect(plugin).To(BeNil())
		})
	})

	Context("Filter", func() {
		var (
			plugin types.FilterPlugin
		)

		BeforeEach(func() {
			hclData := `
mask_len4 = 24
mask_len6 = 64
query_address_enabled = true
response_address_enabled = true
`

			var err error
			plugin, err = mask.Setup(testtool.MustFilterBlock("mask", "test_mask", hclData))
			Expect(err).To(Succeed())
		})

		It("should mask IPv4 addresses correctly", func() {
			dt := &dnstap.Dnstap{
				Type: dnstap.Dnstap_MESSAGE.Enum(),
				Message: &dnstap.Message{
					SocketFamily:    dnstap.SocketFamily_INET.Enum(),
					QueryAddress:    net.IPv4(192, 168, 1, 1).To4(),
					ResponseAddress: net.IPv4(10, 0, 0, 1).To4(),
					Type:            dnstap.Message_AUTH_RESPONSE.Enum(),
					QueryMessage:    dnsMsgBytes,
				},
			}
			msg, err := types.NewDnstapMessageFromDnstap(dt)
			Expect(err).To(Succeed())

			result := plugin.Filter(ctx, msg)
			Expect(result).ToNot(BeNil())
			Expect(result.GetDnstap().Message.QueryAddress).To(Equal([]byte(net.IPv4(192, 168, 1, 0).To4())))
			Expect(result.GetDnstap().Message.ResponseAddress).To(Equal([]byte(net.IPv4(10, 0, 0, 0).To4())))
		})

		It("should mask IPv6 addresses correctly", func() {
			dt := &dnstap.Dnstap{
				Type: dnstap.Dnstap_MESSAGE.Enum(),
				Message: &dnstap.Message{
					SocketFamily:    dnstap.SocketFamily_INET6.Enum(),
					QueryAddress:    net.ParseIP("2001:db8::1"),
					ResponseAddress: net.ParseIP("2001:db8:abcd::1"),
					Type:            dnstap.Message_AUTH_RESPONSE.Enum(),
					QueryMessage:    dnsMsgBytes,
				},
			}
			msg, err := types.NewDnstapMessageFromDnstap(dt)
			Expect(err).To(Succeed())

			result := plugin.Filter(ctx, msg)
			Expect(result).ToNot(BeNil())
			Expect(result.GetDnstap().Message.QueryAddress).To(Equal([]byte(net.ParseIP("2001:db8::"))))
			Expect(result.GetDnstap().Message.ResponseAddress).To(Equal([]byte(net.ParseIP("2001:db8:abcd::"))))
		})
	})
})
