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
package unix_test

import (
	"context"
	"io"
	"net"

	"github.com/mimuret/dtap/v3/pkg/plugin/output/unix"
	"github.com/mimuret/dtap/v3/pkg/testtool"
	"github.com/mimuret/dtap/v3/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/net/nettest"
)

var _ = Describe("output/unix", func() {
	Context("Setup", func() {
		var (
			op  types.OutputPlugin
			err error
		)
		When("Path is an empty", func() {
			BeforeEach(func() {
				op, err = unix.Setup(testtool.MustOutputBlock("unix", "test_unix", ``))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp(`The argument "path" is required`))
			})
		})
		When("valid config", func() {
			BeforeEach(func() {
				op, err = unix.Setup(testtool.MustOutputBlock("unix", "test_unix", `path = "/var/run/dnstap.sock"`))
			})
			It("returns error", func() {
				Expect(err).To(Succeed())
				Expect(op).NotTo(BeNil())
			})
		})
	})
	Context("NewConnect", func() {
		var (
			op   types.OutputPlugin
			p    *unix.Unix
			err  error
			path string
			ln   net.Listener
			conn io.Writer
		)
		BeforeEach(func() {
			path, err = nettest.LocalPath()
			Expect(err).To(Succeed())
			ln, err = net.Listen("unix", path)
			Expect(err).To(Succeed())
			op, err = unix.Setup(testtool.MustOutputBlock("unix", "test_unix", `path = "/var/run/dnstap.sock"`))
			Expect(err).To(Succeed())
			p = op.(*unix.Unix)
		})
		AfterEach(func() {
			ln.Close()
		})
		When("failed to connect", func() {
			BeforeEach(func() {
				conn, err = p.NewConnect(context.Background())
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to connect unix socket"))
			})
		})
		When("valid", func() {
			BeforeEach(func() {
				p.Path = path
				conn, err = p.NewConnect(context.Background())
			})
			It("Succeed", func() {
				Expect(err).To(Succeed())
				Expect(conn).NotTo(BeNil())
			})
		})
	})
})
