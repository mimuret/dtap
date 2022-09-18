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
package metrics_test

import (
	"context"
	_ "embed"
	"fmt"
	"net"
	"time"

	"github.com/goccy/go-json"

	dto "github.com/prometheus/client_model/go"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
	"github.com/mimuret/dnsutils/getter"
	"github.com/mimuret/dnsutils/testtool"
	"github.com/mimuret/dtap/v2/pkg/buffer"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/filter/static"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/protobuf/proto"

	"github.com/mimuret/dtap/v2/pkg/plugin/output/metrics"
	"github.com/mimuret/dtap/v2/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:embed testdata/valid.json
var validJson []byte

//go:embed testdata/filterd.json
var filterdJson []byte

func getCounterValue(rule *metrics.MetricsRule, labels []string) (float64, error) {
	cv := metrics.GetCounterVec(rule)
	if cv == nil {
		return 0, fmt.Errorf("not found counter vec")
	}
	var m = &dto.Metric{}
	if err := cv.WithLabelValues(labels...).Write(m); err != nil {
		return 0, err
	}
	return m.Counter.GetValue(), nil
}

var _ = Describe("output/metrics", func() {
	BeforeEach(func() {
		prometheus.DefaultRegisterer = prometheus.NewRegistry()
	})
	Context("Setup", func() {
		var (
			op  types.OutputPlugin
			err error
		)
		When("type mismatch", func() {
			BeforeEach(func() {
				op, err = metrics.Setup(json.RawMessage(`{"Name": "metrics", "Rules": 0}`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to decode config"))
			})
		})
		When("Rules is an empty", func() {
			BeforeEach(func() {
				op, err = metrics.Setup(json.RawMessage(`{"Name": "metrics"}`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("missing parameter Rules"))
			})
		})
		When("Invalid Rule", func() {
			BeforeEach(func() {
				op, err = metrics.Setup(json.RawMessage(`{"Name": "tcp", "Rules": [{}]}`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("invalid rules No"))
			})
		})
		When("valid config", func() {
			BeforeEach(func() {
				op, err = metrics.Setup(json.RawMessage(validJson))
			})
			It("returns error", func() {
				Expect(err).To(Succeed())
				Expect(op).NotTo(BeNil())
			})
		})
	})
	Context("Metrics", func() {
		var (
			msg        *dns.Msg
			msgRaw     []byte
			dt         *dnstap.Dnstap
			dm         *types.DnstapMessage
			raw        []byte
			op         types.OutputPlugin
			o          *metrics.Metrics
			err        error
			buf        types.Buffer
			ctx        context.Context
			cancelFunc context.CancelFunc
		)
		BeforeEach(func() {
			ctx, cancelFunc = context.WithCancel(context.Background())
			buf = buffer.NewRingBuffer(100, nil, nil)
			msg = &dns.Msg{
				MsgHdr: dns.MsgHdr{
					Id:            0,
					Response:      true,
					Opcode:        dns.OpcodeQuery,
					Authoritative: true,
					Rcode:         dns.RcodeSuccess,
				},
				Question: []dns.Question{
					{Name: "example.jp.", Qclass: dns.ClassINET, Qtype: dns.TypeA},
				},
				Answer: []dns.RR{
					testtool.MustNewRR("example.jp. 300 IN A 127.0.0.1"),
				},
			}
			msgRaw, err = msg.Pack()
			Expect(err).To(Succeed())
			port1024 := uint32(1024)
			port443 := uint32(443)
			now := time.Now()
			sec := uint64(now.Unix())
			nsec := uint32(now.UnixNano())
			dt = &dnstap.Dnstap{
				Identity: []byte("localhost."),
				Version:  []byte("dnstap"),
				Type:     dnstap.Dnstap_MESSAGE.Enum(),
				Message: &dnstap.Message{
					Type:             dnstap.Message_AUTH_RESPONSE.Enum(),
					SocketFamily:     dnstap.SocketFamily_INET.Enum(),
					SocketProtocol:   dnstap.SocketProtocol_DOH.Enum(),
					QueryAddress:     net.IPv4(192, 168, 0, 1).To4(),
					ResponseAddress:  net.IPv4(10, 0, 0, 1).To4(),
					QueryPort:        &port443,
					ResponsePort:     &port1024,
					QueryTimeSec:     nil,
					QueryTimeNsec:    nil,
					QueryMessage:     nil,
					QueryZone:        nil,
					ResponseTimeSec:  &sec,
					ResponseTimeNsec: &nsec,
					ResponseMessage:  msgRaw,
				},
			}
			raw, err = proto.Marshal(dt)
			Expect(err).To(Succeed())
			dm, err = types.NewDnstapMessage(raw)
			Expect(err).To(Succeed())
		})
		When("filterd", func() {
			BeforeEach(func() {
				op, err = metrics.Setup(json.RawMessage(filterdJson))
				Expect(err).To(Succeed())
				o = op.(*metrics.Metrics)
				buf.Write(dm)
				go func() {
					err := op.Start(ctx, buf)
					Expect(err).To(Succeed())
				}()
			})
			AfterEach(func() {
				cancelFunc()
			})
			It("not count", func() {
				Eventually(func() int64 {
					if count, err := getCounterValue(o.Rules[0], nil); err != nil {
						return -1
					} else {
						return int64(count)
					}
				}).Should(Equal(int64(1)))
				Eventually(func() int64 {
					if count, err := getCounterValue(o.Rules[1], nil); err != nil {
						return -1
					} else {
						return int64(count)
					}
				}).Should(Equal(int64(0)))
				Eventually(func() int64 {
					if count, err := getCounterValue(o.Rules[2], []string{"10.0.0.1", "NOERROR"}); err != nil {
						return -1
					} else {
						return int64(count)
					}
				}).Should(Equal(int64(1)))
			})
		})
	})
	Context("MetricsRule", func() {
		var (
			err error
			mr  *metrics.MetricsRule
		)
		Context("Setup", func() {
			BeforeEach(func() {
				mr = &metrics.MetricsRule{}
			})
			When("CounterOps Names is empty", func() {
				BeforeEach(func() {
					err = mr.Setup()
				})
				It("returns error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("failed to register metrics"))
				})
			})
			When("CounterOps duplicate", func() {
				BeforeEach(func() {
					mr.CounterOps.Name = "example"
					m2 := metrics.MetricsRule{CounterOps: mr.CounterOps}
					Expect(m2.Setup()).To(Succeed())
					err = mr.Setup()
				})
				It("returns error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("failed to register metrics"))
				})
			})
			When("valid name", func() {
				BeforeEach(func() {
					mr.CounterOps.Name = "example"
					err = mr.Setup()
				})
				It("returns error", func() {
					Expect(err).To(Succeed())
				})
			})
			Context("DnstalLabels", func() {
				When("empty DnstapLabel.Name", func() {
					BeforeEach(func() {
						mr.DnstapLabels = []metrics.DnstapLabel{
							{
								Name:      "",
								Attribute: getter.GetterMessageFamily,
							},
						}
						err = mr.Setup()
					})
					It("returns error", func() {
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(MatchRegexp("DnstapLabel\\[0\\] missing Name"))
					})
				})
				When("unknown DnstapLabel.Attribute", func() {
					BeforeEach(func() {
						mr.DnstapLabels = []metrics.DnstapLabel{
							{
								Name:      "hoge",
								Attribute: "dummy@",
							},
						}
						err = mr.Setup()
					})
					It("returns error", func() {
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(MatchRegexp("DnstapLabel\\[0\\] unknown Attribute"))
					})
				})
				When("duplicate DnstapLabel.Name", func() {
					When("duplicate dnstap", func() {
						BeforeEach(func() {
							mr.DnstapLabels = []metrics.DnstapLabel{
								{
									Name:      "example",
									Attribute: getter.GetterMessageFamily,
								},
								{
									Name:      "example",
									Attribute: getter.GetterMessageFamily,
								},
							}
							err = mr.Setup()
						})
						It("returns error", func() {
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(MatchRegexp("DnstapLabel\\[1\\] duplicate Name"))
						})
					})
					When("duplicate dnstap and dnsmsg", func() {
						BeforeEach(func() {
							mr.DnstapLabels = []metrics.DnstapLabel{
								{
									Name:      "example",
									Attribute: getter.GetterMessageFamily,
								},
							}
							mr.DnsMsgLabels = []metrics.DnsMsgLabel{
								{
									Name:      "example",
									Attribute: getter.GetterAA,
								},
							}
							err = mr.Setup()
						})
						It("returns error", func() {
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(MatchRegexp("DnsMsgLabel\\[0\\] duplicate Name"))
						})
					})
				})
			})
			Context("DnsMsgLabels", func() {
				When("empty DnsMsgLabel.Name", func() {
					BeforeEach(func() {
						mr.DnsMsgLabels = []metrics.DnsMsgLabel{
							{
								Name:      "",
								Attribute: getter.GetterAA,
							},
						}
						err = mr.Setup()
					})
					It("returns error", func() {
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(MatchRegexp("DnsMsgLabel\\[0\\] missing Name"))
					})
				})
				When("unknown DnstapLabel.Attribute", func() {
					BeforeEach(func() {
						mr.DnsMsgLabels = []metrics.DnsMsgLabel{
							{
								Name:      "hoge",
								Attribute: "dummy@",
							},
						}
						err = mr.Setup()
					})
					It("returns error", func() {
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(MatchRegexp("DnsMsgLabel\\[0\\] unknown Attribute"))
					})
				})
				When("duplicate DnstapLabel.Name", func() {
					When("duplicate dnstap", func() {
						BeforeEach(func() {
							mr.DnsMsgLabels = []metrics.DnsMsgLabel{
								{
									Name:      "example",
									Attribute: getter.GetterAA,
								},
								{
									Name:      "example",
									Attribute: getter.GetterAD,
								},
							}
							err = mr.Setup()
						})
						It("returns error", func() {
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(MatchRegexp("DnsMsgLabel\\[1\\] duplicate Name"))
						})
					})
				})
			})
		})
	})
})
