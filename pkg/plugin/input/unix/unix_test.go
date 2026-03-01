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
	"math"
	"os"

	"github.com/mimuret/dtap/v3/pkg/plugin/input/unix"
	"github.com/mimuret/dtap/v3/pkg/testtool"
	"github.com/mimuret/dtap/v3/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/net/nettest"
)

var _ = Describe("input/unix", func() {
	Context("Setup", func() {
		var (
			path string
			err  error
			p    types.InputPlugin
		)
		BeforeEach(func() {
		})
		When("Path is empty", func() {
			BeforeEach(func() {
				p, err = unix.Setup(testtool.MustInputBlock("unix", "test_unix", ``))
			})
			It("returns error", func() {
				Expect(p).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp(`Missing required argument; The argument "path" is required`))
			})
		})
		When("User is missing", func() {
			BeforeEach(func() {
				path, err = nettest.LocalPath()
				Expect(err).To(Succeed())
				p, err = unix.Setup(testtool.MustInputBlock("unix", "test_unix", `
path = "`+path+`"
user = "missing"
`))
			})
			It("returns error", func() {
				Expect(p).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to lookup user"))
			})
		})
		When("Path is a string and User is empty", func() {
			BeforeEach(func() {
				path, err = nettest.LocalPath()
				Expect(err).To(Succeed())

				p, err = unix.Setup(testtool.MustInputBlock("unix", "test_unix", `
path = "`+path+`"
`))
			})
			It("returns error", func() {
				Expect(err).To(Succeed())
				Expect(p.(*unix.UnixSocket).Path).To(Equal(path))
				Expect(unix.GetUid(p.(*unix.UnixSocket))).To(BeNil())
				Expect(unix.GetGid(p.(*unix.UnixSocket))).To(BeNil())
			})
		})
		When("Path is a string and User exists", func() {
			BeforeEach(func() {
				path, err = nettest.LocalPath()
				Expect(err).To(Succeed())

				p, err = unix.Setup(testtool.MustInputBlock("unix", "test_unix", `
path = "`+path+`"
user = "root"
`))
			})
			It("returns error", func() {
				Expect(err).To(Succeed())
				Expect(p.(*unix.UnixSocket).Path).To(Equal(path))
				Expect(*unix.GetUid(p.(*unix.UnixSocket))).To(Equal(0))
				Expect(*unix.GetGid(p.(*unix.UnixSocket))).To(Equal(0))
			})
		})
	})
	Context("Listen", func() {
		var (
			path string
			err  error
			ip   types.InputPlugin
			p    *unix.UnixSocket
		)
		BeforeEach(func() {
			path, err = nettest.LocalPath()
			Expect(err).To(Succeed())

			ip, err = unix.Setup(testtool.MustInputBlock("unix", "test_unix", `
			path = "`+path+`"
			`))

			p = ip.(*unix.UnixSocket)

			path, err = nettest.LocalPath()
			Expect(err).To(Succeed())
		})
		When("failed to listen", func() {
			BeforeEach(func() {
				p.Path = "/hogehoge/hugahuga"
				err = p.Start(context.Background(), nil)
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to listen"))
			})
		})
		When("failed to chown", func() {
			BeforeEach(func() {
				unix.SetUid(p, math.MinInt)
				unix.SetGid(p, math.MinInt)
				err = p.Start(context.Background(), nil)
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to change owner"))
			})
		})
		When("open socket", func() {
			var (
				ctx    context.Context
				cancel context.CancelFunc
			)
			BeforeEach(func() {
				p.Path = path
				ctx, cancel = context.WithCancel(context.Background())
				go func() {
					err = p.Start(ctx, nil)
					Expect(err).To(Succeed())
				}()
			})
			It("succeed", func() {
				Eventually(func() error {
					_, err = os.Stat(path)
					return err
				}, "1s", "100ms").Should(Succeed())
				cancel()
				Eventually(func() error {
					_, err = os.Stat(path)
					return err
				}, "1s", "100ms").ShouldNot(Succeed())
			})
		})
	})
})
