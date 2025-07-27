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

package types_test

import (
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mimuret/dtap/v3/pkg/types"
)

var _ = Describe("Net", func() {
	var (
		err  error
		bs   []byte
		fnet types.Net
		str  string
	)
	BeforeEach(func() {
		str = ""
		bs = nil
		err = nil
		fnet = types.Net{}
	})
	Context("String", func() {
		When("fnet.IP=nil", func() {
			BeforeEach(func() {
				str = fnet.String()
			})
			It("returns <nil>", func() {
				Expect(str).To(Equal("<nil>"))
			})
		})
		When("net.ip != nil", func() {
			BeforeEach(func() {
				fnet = types.Net{
					IP:           net.ParseIP("192.168.0.1"),
					PrefixLength: 24,
				}
				str = fnet.String()
			})
			It("returns IP.String()/prefix_length", func() {
				Expect(str).To(Equal("192.168.0.1/24"))
			})
		})
	})
	Context("MarshalJSON", func() {
		When("fnet.IP=nil", func() {
			BeforeEach(func() {
				bs, err = fnet.MarshalJSON()
			})
			It("succeed", func() {
				Expect(err).To(Succeed())
			})
			It(`returns "<nil>"`, func() {
				Expect(bs).To(Equal([]byte(`"<nil>"`)))
			})
		})
		When("fnet.IP=IPv4", func() {
			BeforeEach(func() {
				fnet = types.Net{
					IP:           net.ParseIP("192.168.0.1"),
					PrefixLength: 32,
				}
				bs, err = fnet.MarshalJSON()
			})
			It("succeed", func() {
				Expect(err).To(Succeed())
			})
			It(`returns "<nil>"`, func() {
				Expect(bs).To(Equal([]byte(`"192.168.0.1/32"`)))
			})
		})
		When("fnet.IP=IPv6", func() {
			BeforeEach(func() {
				fnet = types.Net{
					IP:           net.ParseIP("2001:db8::1"),
					PrefixLength: 48,
				}
				bs, err = fnet.MarshalJSON()
			})
			It("succeed", func() {
				Expect(err).To(Succeed())
			})
			It(`returns "<nil>"`, func() {
				Expect(bs).To(Equal([]byte(`"2001:db8::1/48"`)))
			})
		})
	})
	Context("UnmarshalJSON", func() {
		When("<nil>", func() {
			BeforeEach(func() {
				fnet = types.Net{
					IP:           net.ParseIP("2001:db8::1"),
					PrefixLength: 48,
				}
				bs = []byte(`"<nil>"`)
				err = fnet.UnmarshalJSON(bs)
			})
			It("succeed", func() {
				Expect(err).To(Succeed())
			})
			It(`set nil vlue`, func() {
				Expect(fnet.IP).To(BeNil())
			})
		})
		When("ipv4", func() {
			BeforeEach(func() {
				bs = []byte(`"192.168.0.1/32"`)
				err = fnet.UnmarshalJSON(bs)
			})
			It("succeed", func() {
				Expect(err).To(Succeed())
			})
			It(`set value`, func() {
				Expect(fnet.IP).To(Equal(net.ParseIP("192.168.0.1")))
				Expect(fnet.PrefixLength).To(Equal(32))
			})
		})
		When("ipv6", func() {
			BeforeEach(func() {
				bs = []byte(`"2001:db8::1/48"`)
				err = fnet.UnmarshalJSON(bs)
			})
			It("succeed", func() {
				Expect(err).To(Succeed())
			})
			It(`set value`, func() {
				Expect(fnet.IP).To(Equal(net.ParseIP("2001:db8::1")))
				Expect(fnet.PrefixLength).To(Equal(48))
			})
		})
		When("invalid ip format", func() {
			var testcases = []string{
				`"999.999.999.0/0"`,
				`"2001:db8:::1/0"`,
				`"192.168.0.0/33"`,
				`"192.168.0.0/-1"`,
				`"2001:db8::1/129"`,
				`"2001:db8::1/-1"`,
			}

			It("failed", func() {
				for _, str := range testcases {
					err = fnet.UnmarshalJSON([]byte(str))
					Expect(err).To(HaveOccurred(), str)
				}
			})
		})
	})
})
