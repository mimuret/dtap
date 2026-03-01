package types_test

import (
	"net"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	"github.com/mimuret/dtap/v3/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("DnstapMessage", func() {
	var (
		raw []byte
		err error
		dm  *types.DnstapMessage
	)
	Context("NewDnstapMessage", func() {
		When("param is nil", func() {
			BeforeEach(func() {
				dm, err = types.NewDnstapMessage(nil)
			})
			It("return nil", func() {
				Expect(err).To(HaveOccurred())
				Expect(dm).To(BeNil())
			})
		})
		When("param is not dnstap raw", func() {
			BeforeEach(func() {
				dm, err = types.NewDnstapMessage([]byte{})
			})
			It("return nil", func() {
				Expect(err).To(HaveOccurred())
				Expect(dm).To(BeNil())
			})
		})
		When("param is dnstap raw", func() {
			When("QueryMessage and ResponseMessage are empty", func() {
				BeforeEach(func() {
					dt := &dnstap.Dnstap{
						Identity: []byte("localhost."),
						Version:  []byte("dnstap"),
						Type:     dnstap.Dnstap_MESSAGE.Enum(),
						Message: &dnstap.Message{
							Type:            dnstap.Message_AUTH_RESPONSE.Enum(),
							SocketFamily:    dnstap.SocketFamily_INET.Enum(),
							SocketProtocol:  dnstap.SocketProtocol_DOH.Enum(),
							QueryAddress:    net.IPv4(192, 168, 0, 1).To4(),
							ResponseAddress: net.IPv4(10, 0, 0, 1).To4(),
							ResponseMessage: nil,
							QueryMessage:    nil,
						},
					}
					raw, err = proto.Marshal(dt)
					Expect(err).To(Succeed())
					dm, err = types.NewDnstapMessage(raw)
				})
				It("return nil", func() {
					Expect(err).To(HaveOccurred())
					Expect(dm).To(BeNil())
				})
			})
			When("QueryMessage and ResponseMessage is not DNS message", func() {
				BeforeEach(func() {
					dt := &dnstap.Dnstap{
						Identity: []byte("localhost."),
						Version:  []byte("dnstap"),
						Type:     dnstap.Dnstap_MESSAGE.Enum(),
						Message: &dnstap.Message{
							Type:            dnstap.Message_AUTH_RESPONSE.Enum(),
							SocketFamily:    dnstap.SocketFamily_INET.Enum(),
							SocketProtocol:  dnstap.SocketProtocol_DOH.Enum(),
							QueryAddress:    net.IPv4(192, 168, 0, 1).To4(),
							ResponseAddress: net.IPv4(10, 0, 0, 1).To4(),
							ResponseMessage: []byte{0},
							QueryMessage:    nil,
						},
					}
					raw, err = proto.Marshal(dt)
					Expect(err).To(Succeed())
					dm, err = types.NewDnstapMessage(raw)
				})
				It("return nil", func() {
					Expect(err).To(HaveOccurred())
					Expect(dm).To(BeNil())
				})
			})
			When("QueryMessage is DNS message", func() {
				BeforeEach(func() {
					var msgRaw []byte
					msg := &dns.Msg{}
					msg.SetQuestion("example.jp.", dns.TypeA)
					msgRaw, err = msg.Pack()
					Expect(err).To(Succeed())
					dt := &dnstap.Dnstap{
						Identity: []byte("localhost."),
						Version:  []byte("dnstap"),
						Type:     dnstap.Dnstap_MESSAGE.Enum(),
						Message: &dnstap.Message{
							Type:            dnstap.Message_AUTH_QUERY.Enum(),
							SocketFamily:    dnstap.SocketFamily_INET.Enum(),
							SocketProtocol:  dnstap.SocketProtocol_DOH.Enum(),
							QueryAddress:    net.IPv4(192, 168, 0, 1).To4(),
							ResponseAddress: net.IPv4(10, 0, 0, 1).To4(),
							ResponseMessage: nil,
							QueryMessage:    msgRaw,
						},
					}
					raw, err = proto.Marshal(dt)
					Expect(err).To(Succeed())
					dm, err = types.NewDnstapMessage(raw)
				})
				It("succeed", func() {
					Expect(err).To(Succeed())
					Expect(dm).NotTo(BeNil())
				})
			})
			When("ResponseMessage is DNS message", func() {
				BeforeEach(func() {
					var msgRaw []byte
					msg := &dns.Msg{}
					msg.SetReply(msg)
					msgRaw, err = msg.Pack()
					Expect(err).To(Succeed())
					dt := &dnstap.Dnstap{
						Identity: []byte("localhost."),
						Version:  []byte("dnstap"),
						Type:     dnstap.Dnstap_MESSAGE.Enum(),
						Message: &dnstap.Message{
							Type:            dnstap.Message_AUTH_RESPONSE.Enum(),
							SocketFamily:    dnstap.SocketFamily_INET.Enum(),
							SocketProtocol:  dnstap.SocketProtocol_DOH.Enum(),
							QueryAddress:    net.IPv4(192, 168, 0, 1).To4(),
							ResponseAddress: net.IPv4(10, 0, 0, 1).To4(),
							ResponseMessage: msgRaw,
							QueryMessage:    nil,
						},
					}
					raw, err = proto.Marshal(dt)
					Expect(err).To(Succeed())
					dm, err = types.NewDnstapMessage(raw)
				})
				It("succeed", func() {
					Expect(err).To(Succeed())
					Expect(dm).NotTo(BeNil())
				})
			})
		})
	})
	Context("NewDnstapMessageFromDnstap", func() {
		When("param is nil", func() {
			BeforeEach(func() {
				dm, err = types.NewDnstapMessageFromDnstap(nil)
			})
			It("return nil", func() {
				Expect(err).To(HaveOccurred())
				Expect(dm).To(BeNil())
			})
		})
		When("param is invalid", func() {
			BeforeEach(func() {
				dm, err = types.NewDnstapMessageFromDnstap(&dnstap.Dnstap{})
			})
			It("return nil", func() {
				Expect(err).To(HaveOccurred())
				Expect(dm).To(BeNil())
			})
		})
		When("dnsMsg is invalid", func() {
			BeforeEach(func() {
				dt := &dnstap.Dnstap{
					Identity: []byte("localhost."),
					Version:  []byte("dnstap"),
					Type:     dnstap.Dnstap_MESSAGE.Enum(),
					Message: &dnstap.Message{
						Type:            dnstap.Message_AUTH_RESPONSE.Enum(),
						SocketFamily:    dnstap.SocketFamily_INET.Enum(),
						SocketProtocol:  dnstap.SocketProtocol_DOH.Enum(),
						QueryAddress:    net.IPv4(192, 168, 0, 1).To4(),
						ResponseAddress: net.IPv4(10, 0, 0, 1).To4(),
						ResponseMessage: []byte{10},
						QueryMessage:    nil,
					},
				}
				dm, err = types.NewDnstapMessageFromDnstap(dt)
			})
			It("return nil", func() {
				Expect(err).To(HaveOccurred())
				Expect(dm).To(BeNil())
			})
		})
		When("valid DNSTAP", func() {
			BeforeEach(func() {
				var msgRaw []byte
				msg := &dns.Msg{}
				msg.SetReply(msg)
				msgRaw, err = msg.Pack()
				Expect(err).To(Succeed())
				dt := &dnstap.Dnstap{
					Identity: []byte("localhost."),
					Version:  []byte("dnstap"),
					Type:     dnstap.Dnstap_MESSAGE.Enum(),
					Message: &dnstap.Message{
						Type:            dnstap.Message_AUTH_RESPONSE.Enum(),
						SocketFamily:    dnstap.SocketFamily_INET.Enum(),
						SocketProtocol:  dnstap.SocketProtocol_DOH.Enum(),
						QueryAddress:    net.IPv4(192, 168, 0, 1).To4(),
						ResponseAddress: net.IPv4(10, 0, 0, 1).To4(),
						ResponseMessage: msgRaw,
						QueryMessage:    nil,
					},
				}
				raw, err = proto.Marshal(dt)
				Expect(err).To(Succeed())
				dm, err = types.NewDnstapMessage(raw)
			})
			It("succeed", func() {
				Expect(err).To(Succeed())
				Expect(dm).NotTo(BeNil())
			})
		})
	})
	Context("NewDnstapMessageFromDnstap", func() {
		BeforeEach(func() {
			var msgRaw []byte
			msg := &dns.Msg{}
			msg.SetReply(msg)
			msgRaw, err = msg.Pack()
			Expect(err).To(Succeed())
			dt := &dnstap.Dnstap{
				Identity: []byte("localhost."),
				Version:  []byte("dnstap"),
				Type:     dnstap.Dnstap_MESSAGE.Enum(),
				Message: &dnstap.Message{
					Type:            dnstap.Message_AUTH_RESPONSE.Enum(),
					SocketFamily:    dnstap.SocketFamily_INET.Enum(),
					SocketProtocol:  dnstap.SocketProtocol_DOH.Enum(),
					QueryAddress:    net.IPv4(192, 168, 0, 1).To4(),
					ResponseAddress: net.IPv4(10, 0, 0, 1).To4(),
					ResponseMessage: msgRaw,
					QueryMessage:    nil,
				},
			}
			raw, err = proto.Marshal(dt)
			Expect(err).To(Succeed())
			dm, err = types.NewDnstapMessage(raw)
			Expect(err).To(Succeed())
		})
		When("params is nil", func() {
			BeforeEach(func() {
				err = dm.UpdateFromDnstap(nil)
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("dnstap is nil"))
			})
		})
		When("params is invalid dnstap", func() {
			BeforeEach(func() {
				err = dm.UpdateFromDnstap(&dnstap.Dnstap{})
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to Marshal dnstap"))
			})
		})
		When("params is invalid dns message", func() {
			BeforeEach(func() {
				err = dm.UpdateFromDnstap(&dnstap.Dnstap{
					Identity: []byte("localhost."),
					Version:  []byte("dnstap"),
					Type:     dnstap.Dnstap_MESSAGE.Enum(),
					Message: &dnstap.Message{
						Type:            dnstap.Message_AUTH_RESPONSE.Enum(),
						SocketFamily:    dnstap.SocketFamily_INET.Enum(),
						SocketProtocol:  dnstap.SocketProtocol_DOH.Enum(),
						QueryAddress:    net.IPv4(192, 168, 0, 1).To4(),
						ResponseAddress: net.IPv4(10, 0, 0, 1).To4(),
						ResponseMessage: []byte{0},
					},
				})
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to Unpack dns msg"))
			})
		})
		When("valid dnstap", func() {
			BeforeEach(func() {
				var msgRaw []byte
				msg := &dns.Msg{}
				msg.SetQuestion("example.net.", dns.TypeA)
				msgRaw, err = msg.Pack()
				Expect(err).To(Succeed())
				dt := &dnstap.Dnstap{
					Identity: []byte("localhost."),
					Version:  []byte("dnstap2"),
					Type:     dnstap.Dnstap_MESSAGE.Enum(),
					Message: &dnstap.Message{
						Type:            dnstap.Message_AUTH_RESPONSE.Enum(),
						SocketFamily:    dnstap.SocketFamily_INET6.Enum(),
						SocketProtocol:  dnstap.SocketProtocol_DOH.Enum(),
						QueryAddress:    net.IPv4(192, 168, 0, 1).To4(),
						ResponseAddress: net.IPv4(10, 0, 0, 1).To4(),
						ResponseMessage: msgRaw,
						QueryMessage:    nil,
					},
				}
				raw, err = proto.Marshal(dt)
				Expect(err).To(Succeed())
				err = dm.UpdateFromDnstap(dt)
			})
			It("succeed", func() {
				Expect(err).To(Succeed())
				Expect(dm.GetDnstap().GetVersion()).To(Equal([]byte("dnstap2")))
				Expect(dm.GetDnstap().GetMessage().GetSocketFamily()).To(Equal(dnstap.SocketFamily_INET6))
				Expect(dm.GetMessage().Question[0].Name).To(Equal("example.net."))
			})
		})
	})
	Context("DeepCopy", func() {
		var (
			dm2 *types.DnstapMessage
		)
		BeforeEach(func() {
			var msgRaw []byte
			msg := &dns.Msg{}
			msg.SetQuestion("example.jp.", dns.TypeA)
			msg.SetReply(msg)
			msgRaw, err = msg.Pack()
			Expect(err).To(Succeed())
			dt := &dnstap.Dnstap{
				Identity: []byte("localhost."),
				Version:  []byte("dnstap"),
				Type:     dnstap.Dnstap_MESSAGE.Enum(),
				Message: &dnstap.Message{
					Type:            dnstap.Message_AUTH_QUERY.Enum().Enum(),
					SocketFamily:    dnstap.SocketFamily_INET.Enum(),
					SocketProtocol:  dnstap.SocketProtocol_DOH.Enum(),
					QueryAddress:    net.IPv4(192, 168, 0, 1).To4(),
					ResponseAddress: net.IPv4(10, 0, 0, 1).To4(),
					QueryMessage:    msgRaw,
				},
			}
			raw, err = proto.Marshal(dt)
			Expect(err).To(Succeed())
			dm, err = types.NewDnstapMessage(raw)
			Expect(err).To(Succeed())
			dm2 = dm.DeepCopy()
		})
		It("returns copy", func() {
			Expect(proto.Equal(dm.GetDnstap(), dm2.GetDnstap())).To(BeTrue())
			Expect(dm2.GetRaw()).To(Equal(dm.GetRaw()))
		})
	})
})
