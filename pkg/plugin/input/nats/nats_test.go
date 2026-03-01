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
	"time"

	"github.com/mimuret/dtap/v3/pkg/plugin/input/nats"
	"github.com/mimuret/dtap/v3/pkg/testtool"
	"github.com/mimuret/dtap/v3/pkg/types"
	"github.com/nats-io/nats-server/v2/server"
	natsio "github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
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
		When("vaild", func() {
			BeforeEach(func() {
				p, err = nats.Setup(testtool.MustInputBlock("nats", "test_nats", `
hosts = ["127.0.0.1:4222"]
subject = "dnstap"
format = "DNSTAP"
`))
			})
			It("returns error", func() {
				Expect(err).To(Succeed())
				Expect(p).NotTo(BeNil())
			})
		})
		When("Host is an empty", func() {
			BeforeEach(func() {
				p, err = nats.Setup(testtool.MustInputBlock("nats", "test_nats", `
hosts = []
subject = "dnstap"
format = "DNSTAP"
`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("missing parameter hosts"))
			})
		})
		When("Subject is an empty", func() {
			BeforeEach(func() {
				p, err = nats.Setup(testtool.MustInputBlock("nats", "test_nats", `
hosts = ["127.0.0.1:4222"]
format = "DNSTAP"
`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp(`The argument "subject" is required`))
			})
		})
		When("Format is invalid", func() {
			BeforeEach(func() {
				p, err = nats.Setup(testtool.MustInputBlock("nats", "test_nats", `
hosts = ["127.0.0.1:4222"]
subject = "dnstap"
format = "hoge"
`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("invalid format"))
			})
		})
		When("Token exist", func() {
			BeforeEach(func() {
				p, err = nats.Setup(testtool.MustInputBlock("nats", "test_nats", `
hosts = ["127.0.0.1:4222"]
subject = "dnstap"
format = "dnstap"
token = "hoge"
`))
			})
			It("succeed", func() {
				Expect(err).To(Succeed())
			})
		})
		When("Token not exist", func() {
			When("User exist", func() {
				When("Password exist", func() {
					BeforeEach(func() {
						p, err = nats.Setup(testtool.MustInputBlock("nats", "test_nats", `
						hosts = ["127.0.0.1:4222"]
						subject = "dnstap"
						format = "dnstap"
						user = "user"
						password = "pass"
						`))
					})
					It("succeed", func() {
						Expect(err).To(Succeed())
					})
				})
				When("Password not exist", func() {
					BeforeEach(func() {
						p, err = nats.Setup(testtool.MustInputBlock("nats", "test_nats", `
						hosts = ["127.0.0.1:4222"]
						subject = "dnstap"
						format = "dnstap"
						user = "user"
						`))
					})
					It("returns error", func() {
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(MatchRegexp("missing parameter password"))
					})
				})
			})
			When("User not exist", func() {
				When("Password exist", func() {
					BeforeEach(func() {
						p, err = nats.Setup(testtool.MustInputBlock("nats", "test_nats", `
						hosts = ["127.0.0.1:4222"]
						subject = "dnstap"
						format = "dnstap"
						password = "password"
						`))
					})
					It("succeed", func() {
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(MatchRegexp("missing parameter user"))
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
					inp, err = nats.Setup(testtool.MustInputBlock("nats", "test_nats", `
hosts = ["127.0.0.1:5223"]
subject = "dnstap"
format = "DNSTAP"
`))
					Expect(err).To(Succeed())
					p = inp.(*nats.Nats)
					_, err = p.Open()
				})
				It("returns error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("failed to connect to NATS server"))
				})
			})
			When("valid", func() {
				BeforeEach(func() {
					inp, err = nats.Setup(testtool.MustInputBlock("nats", "test_nats", `
					hosts = ["127.0.0.1:14223"]
					subject = "dnstap"
					format = "DNSTAP"
					`))
					Expect(err).To(Succeed())
					p = inp.(*nats.Nats)
					_, err = p.Open()
				})
				It("Succeed", func() {
					Expect(err).To(Succeed())
				})
			})
		})
		Context("Start", func() {
			var (
				ctx          context.Context
				cancelFunc   context.CancelFunc
				forwader     *testtool.TestForwader
				subscribeErr error
			)
			BeforeEach(func() {
				ctx, cancelFunc = context.WithCancel(context.Background())
				forwader = &testtool.TestForwader{}
			})
			When("valid message", func() {
				When("Format is DNSTAP", func() {
					BeforeEach(func() {
						inp, err = nats.Setup(testtool.MustInputBlock("nats", "test_nats", `
						hosts = ["127.0.0.1:14223"]
						subject = "dnstap"
						format = "DNSTAP"
						`))
						Expect(err).To(Succeed())
						p = inp.(*nats.Nats)
					})
					BeforeEach(func() {
						go func() {
							subscribeErr = p.Start(ctx, forwader)
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
							err = cl.Publish("dnstap", dnstapRaw)
							Expect(err).To(Succeed())
						})
						It("succeed", func() {
							Eventually(func() int { return forwader.Forwarded }, time.Second*5).Should(Equal(12))
						})
					})
				})
			})
		})
	})
})
