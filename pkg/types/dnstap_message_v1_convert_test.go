package types_test

import (
	"encoding/json"

	"github.com/miekg/dns"
	"github.com/mimuret/dnsutils"
	"github.com/mimuret/dtap/v2/pkg/testtool"
	"github.com/mimuret/dtap/v2/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DnstapMessage", func() {
	var (
		res map[string]interface{}
		dm  *types.DnstapMessage
	)
	Context("ConvertV1JSONWithFilter", func() {
		BeforeEach(func() {
			dm = testtool.CreateValidDnstapMessage()
			res = map[string]interface{}{}
		})
		When("With OutputFilters", func() {
			BeforeEach(func() {
				jsonraw, err := dm.ConvertV1JSONWithFilter(types.OutputFilters{IncludeKeys: []string{"qname", "qclass", "dummy"}})
				Expect(err).To(Succeed())
				err = json.Unmarshal(jsonraw, &res)
				Expect(err).To(Succeed())
			})
			It("return only includeKey", func() {
				Expect(res["qname"]).To(Equal(dm.GetMessage().Question[0].Name))
				Expect(res["qclass"]).To(Equal(dnsutils.ConvertClassToString(dns.Class(dm.GetMessage().Question[0].Qclass))))
				Expect(res).NotTo(HaveKey("dummy"))
			})
		})
		When("With excludeKeys", func() {
			BeforeEach(func() {
				jsonraw, err := dm.ConvertV1JSONWithFilter(types.OutputFilters{ExcludeKeys: []string{"qname", "qclass", "dummy"}})
				Expect(err).To(Succeed())
				err = json.Unmarshal(jsonraw, &res)
				Expect(err).To(Succeed())
			})
			It("return without excludeKey", func() {
				Expect(res).NotTo(HaveKey("qname"))
				Expect(res).NotTo(HaveKey("qclass"))
				Expect(res).To(HaveKey("qtype"))
				Expect(res).NotTo(HaveKey("dummy"))
			})
		})
		When("With includeKeys and excludeKeys", func() {
			BeforeEach(func() {
				jsonraw, err := dm.ConvertV1JSONWithFilter(types.OutputFilters{IncludeKeys: []string{"qname", "qclass", "dummy"}, ExcludeKeys: []string{"qclass", "dummy"}})
				Expect(err).To(Succeed())
				err = json.Unmarshal(jsonraw, &res)
				Expect(err).To(Succeed())
			})
			It("return without excludeKey", func() {
				Expect(res).To(HaveKey("qname"))
				Expect(res).NotTo(HaveKey("qclass"))
				Expect(res).NotTo(HaveKey("qtype"))
				Expect(res).NotTo(HaveKey("dummy"))
			})
		})
	})
})
