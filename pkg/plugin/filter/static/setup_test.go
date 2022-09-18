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
package static_test

import (
	_ "embed"

	"github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/filter/static"
	"github.com/mimuret/dtap/v2/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Static", func() {
	Context("Setup", func() {
		var (
			err error
			fp  types.FilterPlugin
		)
		When("invalid json", func() {
			BeforeEach(func() {
				fp, err = static.Setup(json.RawMessage(`{"Name": "static", "Deny": 1}`))
			})
			It("returns static", func() {
				Expect(err).To(HaveOccurred())
			})
		})
		When("valid json", func() {
			BeforeEach(func() {
				fp, err = static.Setup(json.RawMessage(`{"Name": "static", "Deny": true}`))
			})
			It("returns static", func() {
				Expect(err).To(Succeed())
				Expect(fp).To(Equal(&static.Static{PluginCommon: plugin.PluginCommon{Name: "static"}, Deny: true}))
			})
		})
	})
	Context("Filter", func() {
		var (
			fp  *static.Static
			dm1 *types.DnstapMessage
			dm2 *types.DnstapMessage
		)
		BeforeEach(func() {
			fp = &static.Static{}
			dm1 = &types.DnstapMessage{}
		})
		When("Deny is false", func() {
			BeforeEach(func() {
				fp.Deny = false
				dm2 = fp.Filter(dm1)
			})
			It("through", func() {
				Expect(dm2).To(Equal(dm1))
			})
		})
		When("Deny is true", func() {
			BeforeEach(func() {
				fp.Deny = true
				dm2 = fp.Filter(dm1)
			})
			It("filtered", func() {
				Expect(dm2).To(BeNil())
			})
		})
	})
})
