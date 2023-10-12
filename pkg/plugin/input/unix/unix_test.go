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
	"math"
	"os"

	"github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/plugin/input/unix"
	"github.com/mimuret/dtap/v2/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/nettest"
)

var _ = Describe("input/unix", func() {
	Context("SetupUnixSocket", func() {
		var (
			path string
			err  error
			p    types.InputPlugin
		)
		BeforeEach(func() {
		})
		When("Path is an empty", func() {
			BeforeEach(func() {
				p, err = unix.SetupUnixSocket(json.RawMessage(`{"Name": "unix", "ID": "id1"}`))
			})
			It("returns error", func() {
				Expect(p).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("missing parameter Path"))
			})
		})
		When("Path is not string", func() {
			BeforeEach(func() {
				p, err = unix.SetupUnixSocket(json.RawMessage(`{"Name": "unix", "ID": "id2", "Path": 0}`))
			})
			It("returns error", func() {
				Expect(p).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to decode config"))
			})
		})
		When("User missing", func() {
			BeforeEach(func() {
				path, err = nettest.LocalPath()
				Expect(err).To(Succeed())
				p, err = unix.SetupUnixSocket(json.RawMessage(`{"Name":"unix","ID":"id3","Path":"` + path + `","User":"missing"}`))
			})
			It("returns error", func() {
				Expect(p).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to get owner name"))
			})
		})
		When("Path is string, User empty", func() {
			BeforeEach(func() {
				path, err = nettest.LocalPath()
				Expect(err).To(Succeed())
				p, err = unix.SetupUnixSocket(json.RawMessage(`{"Name":"unix","ID":"id4","Path":"` + path + `"}`))
			})
			It("returns error", func() {
				Expect(err).To(Succeed())
				Expect(p.(*unix.UnixSocket).Path).To(Equal(path))
				Expect(unix.GetUid(p.(*unix.UnixSocket))).To(BeNil())
				Expect(unix.GetGid(p.(*unix.UnixSocket))).To(BeNil())
			})
		})
		When("Path is string, User exist", func() {
			BeforeEach(func() {
				path, err = nettest.LocalPath()
				Expect(err).To(Succeed())
				p, err = unix.SetupUnixSocket(json.RawMessage(`{"Name":"unix","ID":"id5","Path":"` + path + `","User":"root"}`))
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
			ip, err = unix.SetupUnixSocket(json.RawMessage(`{"Name":"unix","ID":"id6","Path":"` + path + `"}`))
			p = ip.(*unix.UnixSocket)

			path, err = nettest.LocalPath()
			Expect(err).To(Succeed())
		})
		When("failed to listen", func() {
			BeforeEach(func() {
				p.Path = "/hogehoge/hugahuga"
				err = p.Listen()
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
				err = p.Listen()
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to change owner"))
			})
		})
		When("open socket", func() {
			BeforeEach(func() {
				err = p.Listen()
			})
			AfterEach(func() {
				p.Close()
				_, err = os.Stat(path)
				Expect(os.IsNotExist(err)).To(BeTrue())
			})
			It("succeed", func() {
				Expect(err).To(Succeed())
			})
		})
	})
})
