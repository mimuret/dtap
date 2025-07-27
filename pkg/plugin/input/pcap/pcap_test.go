//go:build linux

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
package pcap_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/plugin/input/pcap"
	"github.com/mimuret/dtap/v3/pkg/testtool"
	"github.com/mimuret/dtap/v3/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("input/pcap", func() {
	var (
		ifs    []net.Interface
		ifName string
	)
	BeforeEach(func() {
		ifs, _ = net.Interfaces()
		ifName = ifs[0].Name
		if os.Getenv("DTAP_TEST_PCAP_IFNAME") != "" {
			ifName = os.Getenv("DTAP_TEST_PCAP_IFNAME")
		}
	})
	Context("Setup", func() {
		var (
			err error
			p   types.InputPlugin
		)
		When("Interface is an empty", func() {
			BeforeEach(func() {
				p, err = pcap.Setup(testtool.MustInputBlock("pcap", "test_pcap", ``))
			})
			It("returns error", func() {
				Expect(p).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("missing parameter Device"))
			})
		})
		When("Device missing", func() {
			BeforeEach(func() {
				p, err = pcap.Setup(testtool.MustInputBlock("pcap", "test_pcap", `device = "missing"`))
			})
			It("returns error", func() {
				Expect(p).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("missing device"))
			})
		})
		When("Valid config ", func() {
			BeforeEach(func() {
				p, err = pcap.Setup(testtool.MustInputBlock("pcap", "test_pcap", `
device = "`+ifName+`"
bpf = "udp and port 53"
`))
			})
			It("returns error", func() {
				Expect(err).To(Succeed())
				Expect(p.(*pcap.PCAP).Device).To(Equal(ifName))
				Expect(p.(*pcap.PCAP).BPF).To(Equal("udp and port 53"))
			})
		})
	})
	Context("Listen", Label("privileged"), func() {
		var (
			err error
			ip  types.InputPlugin
			p   *pcap.PCAP

			ctx        context.Context
			cancelFunc context.CancelFunc
		)
		BeforeEach(func() {
			ip, err = pcap.Setup(testtool.MustInputBlock("pcap", "test_pcap", `
			device = "`+ifName+`"
			`))
			p = ip.(*pcap.PCAP)
		})
		When("open socket", func() {
			BeforeEach(func() {
				ctx, cancelFunc = context.WithCancel(context.Background())
				err = fmt.Errorf("dummy")
				wg := sync.WaitGroup{}
				wg.Add(1)
				go func() {
					err = p.Start(ctx, &plugin.Forwarder{})
					wg.Done()
				}()
				cancelFunc()
				wg.Wait()
			})
			It("succeed", func() {
				Expect(err).To(Succeed())
			})
		})
	})
})
