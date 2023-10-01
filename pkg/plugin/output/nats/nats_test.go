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
package nats_test

import (
	"time"

	"github.com/goccy/go-json"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/mimuret/dtap/v2/pkg/plugin/output/nats"
	"github.com/mimuret/dtap/v2/pkg/plugin/pub"
	"github.com/mimuret/dtap/v2/pkg/testtool"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/nats-io/nats-server/v2/server"
	natsio "github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("output/nats", func() {
	Context("Setup", func() {
		var (
			op  types.OutputPlugin
			err error
		)
		When("type mismatch", func() {
			BeforeEach(func() {
				op, err = nats.Setup(json.RawMessage(`{"Name": "nats","ID":"id1","Hosts": 0}`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to decode config"))
			})
		})
		When("vaild", func() {
			BeforeEach(func() {
				op, err = nats.Setup(json.RawMessage(`{"Name": "nats", "ID":"id2","Hosts": ["127.0.0.1:4222"], "Subject": "dnstap", "Format": "json/v1"}`))
			})
			It("returns error", func() {
				Expect(err).To(Succeed())
				Expect(op).NotTo(BeNil())
			})
		})
		When("Host is an empty", func() {
			BeforeEach(func() {
				op, err = nats.Setup(json.RawMessage(`{"Name": "nats", "ID":"id3","Subject": "dnstap", "Format": "json/v1"}`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("missing parameter Hosts"))
			})
		})
		When("Subject is an empty", func() {
			BeforeEach(func() {
				op, err = nats.Setup(json.RawMessage(`{"Name": "nats", "ID":"id4","Hosts": ["127.0.0.1:4222"],  "Format": "json/v1"}`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("missing parameter Subject"))
			})
		})
		When("Format is empty", func() {
			BeforeEach(func() {
				op, err = nats.Setup(json.RawMessage(`{"Name": "nats", "ID":"id5","Subject": "dnstap", "Hosts": ["127.0.0.1:4222"], "Format": ""}`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to create publisher"))
			})
		})
		When("Format is invalid", func() {
			BeforeEach(func() {
				op, err = nats.Setup(json.RawMessage(`{"Name": "nats", "ID":"id6","Subject": "dnstap", "Hosts": ["127.0.0.1:4222"],  "Format": "hoge"}`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to create publisher"))
			})
		})
		When("Token exist", func() {
			BeforeEach(func() {
				op, err = nats.Setup(json.RawMessage(`{"Name": "nats", "ID":"id7","Subject": "dnstap", "Hosts": ["127.0.0.1:4222"], "Subject": "dnstap", "Format": "json/v1","Token": "token"}`))
			})
			It("succeed", func() {
				Expect(err).To(Succeed())
			})
		})
		When("Token not exist", func() {
			When("User exist", func() {
				When("Password exist", func() {
					BeforeEach(func() {
						op, err = nats.Setup(json.RawMessage(`{"Name": "nats", "ID":"id8", "Subject": "dnstap", "Hosts": ["127.0.0.1:4222"], "Subject": "dnstap", "Format": "json/v1","User":"user","Password":"pass"}`))
					})
					It("succeed", func() {
						Expect(err).To(Succeed())
					})
				})
				When("Password not exist", func() {
					BeforeEach(func() {
						op, err = nats.Setup(json.RawMessage(`{"Name": "nats", "ID":"id9", "Subject": "dnstap", "Hosts": ["127.0.0.1:4222"], "Subject": "dnstap", "Format": "json/v1","User":"user"}`))
					})
					It("returns error", func() {
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(MatchRegexp("missing parameter Password"))
					})
				})
			})
			When("User not exist", func() {
				When("Password exist", func() {
					BeforeEach(func() {
						op, err = nats.Setup(json.RawMessage(`{"Name": "nats", "ID":"id10", "Subject": "dnstap", "Hosts": ["127.0.0.1:4222"], "Subject": "dnstap", "Format": "json/v1","Password":"pass"}`))
					})
					It("succeed", func() {
						Expect(err).To(Succeed())
					})
				})
			})
		})
	})
	Context("Nats", func() {
		var (
			op  types.OutputPlugin
			p   *nats.Nats
			err error
			sv  *server.Server

			nc   *natsio.Conn
			sub  *natsio.Subscription
			ch   chan *natsio.Msg
			data []byte
		)
		BeforeEach(func() {
			ch = make(chan *natsio.Msg)
			sv, err = server.NewServer(&server.Options{
				Host:       "127.0.0.1",
				Port:       14222,
				HTTPPort:   -1,
				Cluster:    server.ClusterOpts{Port: -1, Name: "abc"},
				NoLog:      true,
				NoSigs:     true,
				Debug:      true,
				Trace:      true,
				MaxPayload: server.MAX_PAYLOAD_SIZE,
			})

			Expect(err).To(Succeed())
			Expect(sv).NotTo(BeNil())
			go sv.Start()
			Expect(sv.ReadyForConnections(time.Second * 5)).To(BeTrue())

			nc, err = natsio.Connect("127.0.0.1:14222")
			Expect(err).To(Succeed())
			sub, err = nc.ChanQueueSubscribe("dnstap", "", ch)
			Expect(err).To(Succeed())
			prometheus.DefaultRegisterer = prometheus.NewRegistry()
		})
		AfterEach(func() {
			err := sub.Unsubscribe()
			Expect(err).To(Succeed())
			nc.Close()
			sv.Shutdown()
			sv.WaitForShutdown()
		})
		Context("Open", func() {
			var (
				err error
			)
			When("failed to connect", func() {
				BeforeEach(func() {
					op, err = nats.Setup(json.RawMessage(`{"Name": "nats", "ID":"id11", "Hosts": ["127.0.0.1:15222"], "Subject": "dnstap", "Format": "json/v1"}`))
					Expect(err).To(Succeed())
					p = op.(*nats.Nats)
					err = p.Open()
					Expect(err).To(HaveOccurred())
				})
				It("returns error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("failed to create nats producer"))
				})
			})
			When("valid", func() {
				BeforeEach(func() {
					op, err = nats.Setup(json.RawMessage(`{"Name": "nats", "ID":"id12", "Hosts": ["127.0.0.1:14222"], "Subject": "dnstap", "Format": "json/v1"}`))
					Expect(err).To(Succeed())
					p = op.(*nats.Nats)
					err = p.Open()
				})
				It("Succeed", func() {
					Expect(err).To(Succeed())
				})
			})
		})
		Context("Write", func() {
			var (
				dm  *types.DnstapMessage
				err error
			)
			BeforeEach(func() {
				op, err = nats.Setup(json.RawMessage(`{"Name": "nats", "ID":"id13", "Hosts": ["127.0.0.1:14222"], "Subject": "dnstap", "Format": "json/v1"}`))
				Expect(err).To(Succeed())
				p = op.(*nats.Nats)
			})
			When("valid message", func() {
				BeforeEach(func() {
					dm = testtool.CreateValidDnstapMessage()
				})
				When("Format is json/v1", func() {
					BeforeEach(func() {
						op, err = nats.Setup(json.RawMessage(`{"Name": "nats", "ID":"id14", "Hosts": ["127.0.0.1:14222"], "Subject": "dnstap", "Format": "json/v1"}`))
						Expect(err).To(Succeed())
						p = op.(*nats.Nats)
					})
					BeforeEach(func() {
						Expect(p.Open()).To(Succeed())
					})
					AfterEach(func() {
						p.Close()
					})
					When("write messages ", func() {
						BeforeEach(func() {
							p.Format = pub.FormatV1JSON
							data, err = dm.ConvertV1JSON()
							Expect(err).To(Succeed())
							maxMsg := (nats.DefaultMaxPayloadSize - 2) / (len(data) + 1)
							for i := 0; i < maxMsg; i++ {
								err = p.Write(dm)
								Expect(err).To(Succeed())
							}
						})
						It("succeed", func() {
							Expect(err).To(Succeed())
							Expect(nats.GetStatics(p).OutMsgs).To(Equal(uint64(0)))
						})
					})
					When("1 publish", func() {
						BeforeEach(func() {
							p.Format = pub.FormatV1JSON
							data, err = dm.ConvertV1JSON()
							Expect(err).To(Succeed())
							maxMsg := (nats.DefaultMaxPayloadSize-2)/(len(data)+1) + 1
							for i := 0; i < maxMsg; i++ {
								err = p.Write(dm)
								Expect(err).To(Succeed())
							}
						})
						It("succeed", func() {
							Expect(err).To(Succeed())
							Expect(nats.GetStatics(p).OutMsgs).To(Equal(uint64(1)))
							Eventually(func() *natsio.Msg { return <-ch }).ShouldNot(BeNil())
						})
					})
					When("2 publish", func() {
						BeforeEach(func() {
							p.Format = pub.FormatV1JSON
							data, err = dm.ConvertV1JSON()
							Expect(err).To(Succeed())
							maxMsg := (nats.DefaultMaxPayloadSize-2)/(len(data)+1)*2 + 2
							for i := 0; i < maxMsg; i++ {
								err = p.Write(dm)
								Expect(err).To(Succeed())
							}
						})
						It("succeed", func() {
							Expect(err).To(Succeed())
							Expect(nats.GetStatics(p).OutMsgs).To(Equal(uint64(2)))
							Eventually(func() *natsio.Msg { return <-ch }).ShouldNot(BeNil())
						})
					})
				})
				When("Format is DNSTAP", func() {
					BeforeEach(func() {
						op, err = nats.Setup(json.RawMessage(`{"Name": "nats", "ID":"id15", "Hosts": ["127.0.0.1:14222"], "Subject": "dnstap", "Format": "dnstap"}`))
						Expect(err).To(Succeed())
						p = op.(*nats.Nats)
					})
					BeforeEach(func() {
						Expect(p.Open()).To(Succeed())
					})
					AfterEach(func() {
						p.Close()
					})
					When("write one message ", func() {
						BeforeEach(func() {
							p.Format = pub.FormatDNSTAP
							err = p.Write(dm)
							Expect(err).To(Succeed())
							err = nats.GetPublisher(p).Publish()
							Expect(err).To(Succeed())
						})
						It("succeed", func() {
							Expect(err).To(Succeed())
							Expect(nats.GetStatics(p).OutMsgs).To(Equal(uint64(1)))
						})
					})
					When("write messages ", func() {
						BeforeEach(func() {
							p.Format = pub.FormatDNSTAP
							data := dm.GetRaw()
							Expect(err).To(Succeed())
							maxMsg := (nats.DefaultMaxPayloadSize - pub.DnstapFstrmControlHeaderSize*2) / (len(data) + 4)
							for i := 0; i < maxMsg; i++ {
								err = p.Write(dm)
								Expect(err).To(Succeed())
							}
						})
						It("succeed", func() {
							Expect(err).To(Succeed())
							Expect(nats.GetStatics(p).OutMsgs).To(Equal(uint64(0)))
						})
					})
					When("1 publish", func() {
						BeforeEach(func() {
							sv.ConfigureLogger()
							p.Format = pub.FormatDNSTAP
							data := dm.GetRaw()
							Expect(err).To(Succeed())
							maxMsg := (nats.DefaultMaxPayloadSize-pub.DnstapFstrmControlHeaderSize*2)/(len(data)+4) + 1
							for i := 0; i < maxMsg; i++ {
								err = p.Write(dm)
								Expect(err).To(Succeed())
							}
						})
						It("succeed", func() {
							Expect(err).To(Succeed())
							Expect(nats.GetStatics(p).OutMsgs).To(Equal(uint64(1)))
							Eventually(func() *natsio.Msg { return <-ch }).ShouldNot(BeNil())
						})
					})
					When("2 publish", func() {
						BeforeEach(func() {
							p.Format = pub.FormatDNSTAP
							data := dm.GetRaw()
							Expect(err).To(Succeed())
							maxMsg := (nats.DefaultMaxPayloadSize-pub.DnstapFstrmControlHeaderSize*2)/(len(data)+4)*2 + 2
							for i := 0; i < maxMsg; i++ {
								err = p.Write(dm)
								Expect(err).To(Succeed())
							}
						})
						It("succeed", func() {
							Expect(err).To(Succeed())
							Expect(nats.GetStatics(p).OutMsgs).To(Equal(uint64(2)))
							Eventually(func() *natsio.Msg { return <-ch }).ShouldNot(BeNil())
						})
					})
				})
			})
		})
	})
})
