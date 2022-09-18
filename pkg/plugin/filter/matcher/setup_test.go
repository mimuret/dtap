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
package matcher_test

import (
	_ "embed"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	"github.com/mimuret/dnsutils/matcher"
	"github.com/mimuret/dtap/v2/pkg/plugin"
	pmatcher "github.com/mimuret/dtap/v2/pkg/plugin/filter/matcher"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/proto"
)

//go:embed testdata/success.json
var successJson []byte

//go:embed testdata/invalid.json
var invalidJson []byte

var _ = Describe("Static", func() {
	Context("Setup", func() {
		var (
			err error
			fp  types.FilterPlugin
		)
		When("invalid json", func() {
			BeforeEach(func() {
				fp, err = registry.CreateFilterPlugin("matcher", invalidJson)
			})
			It("returns static", func() {
				Expect(err).To(HaveOccurred())
			})
		})
		When("valid json", func() {
			BeforeEach(func() {
				fp, err = registry.CreateFilterPlugin("matcher", successJson)
			})
			It("returns static", func() {
				staticDnstapFalse, _ := matcher.NewMatchDnstapStatic(false)
				staticDnsFalse, _ := matcher.NewMatchDNSMsgStatic(false)
				staticDnsTrue, _ := matcher.NewMatchDNSMsgStatic(true)
				Expect(err).To(Succeed())
				Expect(fp).To(Equal(&pmatcher.Matcher{
					PluginCommon: plugin.PluginCommon{Name: "matcher"},
					Set: &matcher.MatcherSet{
						Op:             matcher.SetOpOR,
						DnstapMatchers: []matcher.DnstapMatcher{staticDnstapFalse},
						DnsMsgMatchers: []matcher.DnsMsgMatcher{staticDnsFalse},
						SubSets: []*matcher.MatcherSet{
							{
								Op:             matcher.SetOpOR,
								DnsMsgMatchers: []matcher.DnsMsgMatcher{staticDnsFalse, staticDnsTrue},
							},
						},
					}}))
			})
		})
	})
	Context("Filter", func() {
		var (
			fp               *pmatcher.Matcher
			dm1, dm2         *types.DnstapMessage
			staticDnstapTrue matcher.DnstapMatcher
			staticDnsTrue    matcher.DnsMsgMatcher
		)
		BeforeEach(func() {
			staticDnstapTrue, _ = matcher.NewMatchDnstapStatic(true)
			staticDnsTrue, _ = matcher.NewMatchDNSMsgStatic(true)
			fp = &pmatcher.Matcher{
				Set: &matcher.MatcherSet{
					Op:             matcher.SetOpAND,
					DnstapMatchers: []matcher.DnstapMatcher{staticDnstapTrue},
				},
			}
		})
		When("valid dns message", func() {
			BeforeEach(func() {
				msg := &dns.Msg{}
				msg.SetQuestion("www.example.jp.", dns.TypeA)
				qmsg, err := msg.Pack()
				Expect(err).To(Succeed())
				tap := &dnstap.Dnstap{
					Type: dnstap.Dnstap_MESSAGE.Enum(),
					Message: &dnstap.Message{
						Type:         dnstap.Message_CLIENT_QUERY.Enum(),
						SocketFamily: dnstap.SocketFamily_INET6.Enum(),
						QueryMessage: qmsg,
					},
				}
				data, err := proto.Marshal(tap)
				Expect(err).To(Succeed())
				dm1, err = types.NewDnstapMessage(data)
				Expect(err).To(Succeed())
			})
			When("match rule", func() {
				BeforeEach(func() {
					m, _ := matcher.NewMatchDNSMsgQueryName("www.example.jp.")
					fp.Set.DnsMsgMatchers = append(fp.Set.DnsMsgMatchers, staticDnsTrue, m)
					dm2 = fp.Filter(dm1)
				})
				It("through", func() {
					Expect(dm2).To(Equal(dm1))
				})
			})
			When("not match rule", func() {
				BeforeEach(func() {
					m, _ := matcher.NewMatchDNSMsgQueryName("www.example.com")
					fp.Set.DnsMsgMatchers = append(fp.Set.DnsMsgMatchers, staticDnsTrue, m)
					dm2 = fp.Filter(dm1)
				})
				It("filterd", func() {
					Expect(dm2).To(BeNil())
				})
			})
		})
	})
})
