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
package label_test

import (
	_ "embed"

	"github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/plugin/filter/label"
	"github.com/mimuret/dtap/v2/pkg/testtool"
	"github.com/mimuret/dtap/v2/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("label", func() {
	Context("Setup", func() {
		var (
			err error
			fp  types.FilterPlugin
		)
		When("invalid json", func() {
			BeforeEach(func() {
				fp, err = label.Setup(json.RawMessage(`{"Name": 0}`))
			})
			It("returns error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
		When("Add", func() {
			When("Add.Name is empty", func() {
				BeforeEach(func() {
					fp, err = label.Setup(json.RawMessage(`{"Name": "label","Add":[{}]}`))
				})
				It("returns error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("missing parameter Name"))
				})
			})
			When("Add.Value is empty", func() {
				BeforeEach(func() {
					fp, err = label.Setup(json.RawMessage(`{"Name": "label","Add":[{"Name": "hoge"}]}`))
				})
				It("returns error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("missing parameter Value"))
				})
			})
			When("Add.Type is invalid", func() {
				BeforeEach(func() {
					fp, err = label.Setup(json.RawMessage(`{"Name": "label","Add":[{"Name": "hoge","Value": "hoge", "Type": "hoge"}]}`))
				})
				It("returns error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("unknown Type"))
				})
			})
			When("Add.Type is DNSTAP", func() {
				When("unknown GetFunc name", func() {
					BeforeEach(func() {
						fp, err = label.Setup(json.RawMessage(`{"Name": "label","Add":[{"Name": "hoge","Value": "hoge", "Type": "DNSTAP"}]}`))
					})
					It("returns error", func() {
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(MatchRegexp("unknown DNSTAP get func"))
					})
				})
				When("valid GetFunc name", func() {
					BeforeEach(func() {
						fp, err = label.Setup(json.RawMessage(`{"Name": "label","Add":[{"Name": "hoge","Value": "ResponseAddress", "Type": "DNSTAP"}]}`))
					})
					It("succeed", func() {
						Expect(err).To(Succeed())
						Expect(fp).NotTo(BeNil())
					})
				})
			})
			When("Add.Type is DNS", func() {
				When("unknown GetFunc name", func() {
					BeforeEach(func() {
						fp, err = label.Setup(json.RawMessage(`{"Name": "label","Add":[{"Name": "hoge","Value": "hoge", "Type": "DNS"}]}`))
					})
					It("returns error", func() {
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(MatchRegexp("unknown DNS get func"))
					})
				})
				When("valid GetFunc name", func() {
					BeforeEach(func() {
						fp, err = label.Setup(json.RawMessage(`{"Name": "label","Add":[{"Name": "hoge","Value": "QName", "Type": "DNS"}]}`))
					})
					It("succeed", func() {
						Expect(err).To(Succeed())
						Expect(fp).NotTo(BeNil())
					})
				})
			})
			When("Add.Type is STATIC", func() {
				BeforeEach(func() {
					fp, err = label.Setup(json.RawMessage(`{"Name": "label","Add":[{"Name": "hoge","Value": "hoge", "Type": "STATIC"}]}`))
				})
				It("succeed", func() {
					Expect(err).To(Succeed())
					Expect(fp).NotTo(BeNil())
				})
			})
		})

		When("Del", func() {
			When("Del.Name is empty", func() {
				BeforeEach(func() {
					fp, err = label.Setup(json.RawMessage(`{"Name": "label","Del":[{}]}`))
				})
				It("returns error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("missing parameter Name"))
				})
			})
		})
	})
	Context("Filter", func() {
		var (
			err error
			fp  types.FilterPlugin
			dm1 *types.DnstapMessage
			dm2 *types.DnstapMessage
		)
		BeforeEach(func() {
			dm1 = testtool.CreateValidDnstapMessage()
			fp, err = label.Setup(json.RawMessage(`{"Name": "label", "Add": [
				{
					"Name": "LabelResponseAddress",
					"Value": "ResponseAddress",
					"Type": "DNSTAP"
				},
				{
					"Name": "LabelQname",
					"Value": "QName",
					"Type": "DNS"
				},
				{
					"Name": "LabelHoge",
					"Value": "hogehoge",
					"Type": "STATIC"
				},
				{
					"Name": "LabelHoge2",
					"Value": "hogehoge",
					"Type": "STATIC"
				}
			],"Del": [
				{"Name": "LabelHoge2"}
			]}`))
			Expect(err).To(Succeed())
			dm2 = fp.Filter(dm1)
		})
		It("adds labels", func() {
			Expect(dm2.Labels["LabelResponseAddress"]).To(Equal("10.0.0.1"))
			Expect(dm2.Labels["LabelQname"]).To(Equal("example.jp."))
			Expect(dm2.Labels["LabelHoge"]).To(Equal("hogehoge"))
			Expect(dm2.Labels["LabelHoge2"]).To(Equal(""))
		})
	})
})
