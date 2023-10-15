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
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/buffer"
	"github.com/mimuret/dtap/v2/pkg/plugin/input/nats"
	"github.com/mimuret/dtap/v2/pkg/testtool"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/nats-io/nats-server/v2/server"
	natsio "github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:embed testdata/dnstap.fstrm
var dnstapRaw []byte

type counter struct {
	I int
}

func (c *counter) Inc() {
	c.I++
}

var _ = Describe("input/nats", func() {
	Context("Setup", func() {
		var (
			p   types.InputPlugin
			err error
		)
		When("type mismatch", func() {
			BeforeEach(func() {
				p, err = nats.Setup(json.RawMessage(`{"Name":"nats","ID":"id1","Hosts":0}`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to decode config"))
			})
		})
		When("vaild", func() {
			BeforeEach(func() {
				p, err = nats.Setup(json.RawMessage(`{"Name":"nats","ID":"id2","Hosts":["127.0.0.1:4222"],"Subject":"dnstap","Format":"DNSTAP"}`))
			})
			It("returns error", func() {
				Expect(err).To(Succeed())
				Expect(p).NotTo(BeNil())
			})
		})
		When("Host is an empty", func() {
			BeforeEach(func() {
				p, err = nats.Setup(json.RawMessage(`{"Name":"nats","ID":"id3","Subject":"dnstap","Format":"DNSTAP"}`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("missing parameter Hosts"))
			})
		})
		When("Subject is an empty", func() {
			BeforeEach(func() {
				p, err = nats.Setup(json.RawMessage(`{"Name":"nats","ID":"id4","Hosts":["127.0.0.1:4222"],"Format":"DNSTAP"}`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("missing parameter Subject"))
			})
		})
		When("Format is empty", func() {
			BeforeEach(func() {
				p, err = nats.Setup(json.RawMessage(`{"Name":"nats","ID":"id5","Subject":"dnstap","Hosts":["127.0.0.1:4222"],"Format": ""}`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("invalid format"))
			})
		})
		When("Format is invalid", func() {
			BeforeEach(func() {
				p, err = nats.Setup(json.RawMessage(`{"Name":"nats","ID":"id6","Subject":"dnstap","Hosts":["127.0.0.1:4222"],"Format":"hoge"}`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("invalid format"))
			})
		})
		When("Token exist", func() {
			BeforeEach(func() {
				p, err = nats.Setup(json.RawMessage(`{"Name":"nats","ID":"id7","Subject":"dnstap","Hosts":["127.0.0.1:4222"],"Subject":"dnstap","Format":"DNSTAP","Token":"token"}`))
			})
			It("succeed", func() {
				Expect(err).To(Succeed())
			})
		})
		When("Token not exist", func() {
			When("User exist", func() {
				When("Password exist", func() {
					BeforeEach(func() {
						p, err = nats.Setup(json.RawMessage(`{"Name":"nats","ID":"id8","Subject":"dnstap","Hosts":["127.0.0.1:4222"],"Subject":"dnstap","Format":"DNSTAP","User":"user","Password":"pass"}`))
					})
					It("succeed", func() {
						Expect(err).To(Succeed())
					})
				})
				When("Password not exist", func() {
					BeforeEach(func() {
						p, err = nats.Setup(json.RawMessage(`{"Name":"nats","ID":"id9","Subject":"dnstap","Hosts":["127.0.0.1:4222"],"Subject":"dnstap","Format":"DNSTAP","User":"user"}`))
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
						p, err = nats.Setup(json.RawMessage(`{"Name":"nats","ID":"id10","Subject":"dnstap","Hosts":["127.0.0.1:4222"],"Subject":"dnstap","Format":"DNSTAP","Password":"pass"}`))
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
			inp types.InputPlugin
			p   *nats.Nats
			err error
			sv  *server.Server

			nc *natsio.Conn
		)
		BeforeEach(func() {
			sv, err = server.NewServer(&server.Options{
				Host:       "127.0.0.1",
				Port:       14223,
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
		})
		AfterEach(func() {
			nc.Close()
			sv.Shutdown()
			sv.WaitForShutdown()
		})
		Context("Open", func() {
			When("failed to connect", func() {
				BeforeEach(func() {
					inp, err = nats.Setup(json.RawMessage(`{"Name":"nats","ID":"id20","Hosts":["127.0.0.1:5223"],"Subject":"dnstap","Format":"DNSTAP"}`))
					Expect(err).To(Succeed())
					p = inp.(*nats.Nats)
					_, err = p.Open()
				})
				It("returns error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("failed to create nats subscriber"))
				})
			})
			When("valid", func() {
				BeforeEach(func() {
					inp, err = nats.Setup(json.RawMessage(`{"Name": "nats","ID":"id21","Hosts": ["127.0.0.1:14223"], "Subject": "dnstap", "Format": "DNSTAP"}`))
					Expect(err).To(Succeed())
					p = inp.(*nats.Nats)
					_, err = p.Open()
				})
				It("Succeed", func() {
					Expect(err).To(Succeed())
				})
			})
		})
		Context("Subscribe", func() {
			var (
				ctx                    context.Context
				cancelFunc             context.CancelFunc
				w                      *buffer.RingBuffer
				subscribeErr           error
				inCounter, lostCounter *counter
			)
			BeforeEach(func() {
				inCounter = &counter{}
				lostCounter = &counter{}
				ctx, cancelFunc = context.WithCancel(context.Background())
				w = buffer.NewRingBuffer(100, inCounter, lostCounter)
			})
			When("valid message", func() {
				When("Format is DNSTAP", func() {
					BeforeEach(func() {
						inp, err = nats.Setup(json.RawMessage(`{"Name": "nats","ID":"id22","Hosts": ["127.0.0.1:14223"], "Subject": "dnstap", "Format": "DNSTAP"}`))
						Expect(err).To(Succeed())
						p = inp.(*nats.Nats)
					})
					BeforeEach(func() {
						go func() {
							subscribeErr = p.Subscribe(ctx, w, testtool.NewTestInputContext(nil))
						}()
					})
					It("succeed", func() {
						Expect(subscribeErr).To(Succeed())
					})
					AfterEach(func() {
						cancelFunc()
					})
					When("input messages ", func() {
						BeforeEach(func() {
							time.Sleep(time.Second)
							cl, err := p.Open()
							Expect(err).To(Succeed())
							fmt.Println("publish")
							err = cl.Publish("dnstap", dnstapRaw)
							Expect(err).To(Succeed())
						})
						It("succeed", func() {
							Eventually(func() int { return inCounter.I }, time.Second*5).Should(Equal(12))
						})
					})
				})
			})
		})
	})
})
