package expr_test

import (
	"context"
	"net"
	"testing"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	"github.com/mimuret/dtap/v3/pkg/plugin/filter/expr"
	"github.com/mimuret/dtap/v3/pkg/testtool"
	"github.com/mimuret/dtap/v3/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestExpr(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Expr Suite")
}

var _ = Describe("Expr", func() {
	var (
		ctx         context.Context
		msg         *types.DnstapMessage
		dnsMsg      *dns.Msg
		dnsMsgBytes []byte
	)

	BeforeEach(func() {
		var err error
		ctx = context.Background()
		dnsMsg = &dns.Msg{}
		dnsMsg.SetQuestion("example.com.", dns.TypeA)
		dnsMsgBytes, _ = dnsMsg.Pack()
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
		msg, err = types.NewDnstapMessageFromDnstap(dt)
		Expect(err).To(Succeed())

	})

	Context("Setup", func() {
		It("should successfully setup with valid HCL", func() {
			hclData := `
expression =  <<EOF
qtype == "A"
EOF
`
			plugin, err := expr.Setup(testtool.MustFilterBlock("expr", "test_expr", hclData))
			Expect(err).To(Succeed())
			Expect(plugin).ToNot(BeNil())
		})

		It("should return an error for invalid expression", func() {
			hclData := `
expression = "qtype =="
`
			plugin, err := expr.Setup(testtool.MustFilterBlock("expr", "test_expr", hclData))
			Expect(err).To(HaveOccurred())
			Expect(plugin).To(BeNil())
		})
	})

	Context("Filter", func() {
		var (
			plugin types.FilterPlugin
		)
		It("should return the message when the expression evaluates to true", func() {
			hclData := `
expression = "qtype == \"A\""
`

			var err error
			plugin, err = expr.Setup(testtool.MustFilterBlock("expr", "test_expr", hclData))
			Expect(err).To(Succeed())

			result := plugin.Filter(ctx, msg)
			Expect(result).To(Equal(msg))
		})

		It("should return nil when the expression evaluates to false", func() {
			hclData := `
expression = "qtype == \"A\""
`

			var err error
			plugin, err = expr.Setup(testtool.MustFilterBlock("expr", "test_expr", hclData))
			Expect(err).To(Succeed())
			result := plugin.Filter(ctx, msg)
			Expect(result).ToNot(BeNil())
		})

		It("should handle invalid message fields gracefully", func() {
			hclData := `
expression = "qtype == 0"
`
			var err error
			plugin, err = expr.Setup(testtool.MustFilterBlock("expr", "test_expr", hclData))
			Expect(err).To(Succeed())
			result := plugin.Filter(ctx, msg)
			Expect(result).To(BeNil())
		})
	})
})
