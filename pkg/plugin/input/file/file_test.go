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
package file_test

import (
	"context"
	_ "embed"

	"github.com/goccy/go-json"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/mimuret/dtap/v2/pkg/buffer"
	"github.com/mimuret/dtap/v2/pkg/plugin/input/file"
	"github.com/mimuret/dtap/v2/pkg/testtool"
	"github.com/mimuret/dtap/v2/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
)

//go:embed testfile/dummy.data
var dummyData []byte

//go:embed testfile/dnstap.fstrm
var validData []byte

var _ = Describe("input/file", func() {
	Context("SetupFile", func() {
		var (
			err error
			p   types.InputPlugin
		)
		When("invalid json", func() {
			BeforeEach(func() {
				prometheus.DefaultRegisterer = prometheus.NewRegistry()
				p, err = file.SetupFile(json.RawMessage(`{"Name":100,"ID":"id1"}`))
			})
			It("returns error", func() {
				Expect(p).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to decode config"))
			})
		})
		When("Path is an empty", func() {
			BeforeEach(func() {
				prometheus.DefaultRegisterer = prometheus.NewRegistry()
				p, err = file.SetupFile(json.RawMessage(`{"Name":"file","ID":"id2"}`))
			})
			It("returns error", func() {
				Expect(p).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("missing parameter Path"))
			})
		})
		When("vaild config", func() {
			BeforeEach(func() {
				prometheus.DefaultRegisterer = prometheus.NewRegistry()
				p, err = file.SetupFile(json.RawMessage(`{"Name":"file","ID":"id3","Path":"/var/tmp/dump.fstrm"}`))
			})
			It("returns error", func() {
				Expect(err).To(Succeed())
				Expect(p.(*file.File).Path).To(Equal("/var/tmp/dump.fstrm"))
			})
		})
	})
	Context("Start", func() {
		var (
			p   types.InputPlugin
			ic  *types.InputContext
			fp  *file.File
			err error
			fs  afero.Fs
			buf types.Buffer
			f   afero.File
		)
		BeforeEach(func() {
			buf = buffer.NewRingBuffer(10, nil, nil)
			ic = testtool.NewTestInputContext(buf)
			fs = afero.NewMemMapFs()
			prometheus.DefaultRegisterer = prometheus.NewRegistry()
			p, err = file.SetupFile(json.RawMessage(`{"Name":"file","ID":"id4","Path":"/var/tmp/dump.fstrm"}`))
			Expect(err).To(Succeed())
			fp = p.(*file.File)
			file.SetFS(fp, fs)
		})
		When("file not exist", func() {
			BeforeEach(func() {
				err = fp.Start(context.TODO(), ic)
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to open file"))
			})
		})
		When("file read error", func() {
			BeforeEach(func() {
				f, err = fs.Create("/var/tmp/dump.fstrm")
				Expect(err).To(Succeed())
				_, err = f.Write(dummyData)
				Expect(err).To(Succeed())
				f.Close()
				err = fp.Start(context.TODO(), ic)
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("to push message"))
			})
		})
		When("valid data", func() {
			BeforeEach(func() {
				f, err = fs.Create("/var/tmp/dump.fstrm")
				Expect(err).To(Succeed())
				_, err = f.Write(validData)
				Expect(err).To(Succeed())
				f.Close()
				prometheus.DefaultRegisterer = prometheus.NewRegistry()
				err = fp.Start(context.TODO(), ic)
			})
			It("returns error", func() {
				Expect(err).To(Succeed())
				Eventually(func() *types.DnstapMessage {
					return <-buf.Read()
				}).ShouldNot(BeNil())
				Eventually(func() *types.DnstapMessage {
					return <-buf.Read()
				}).ShouldNot(BeNil())
			})
		})
	})
})
