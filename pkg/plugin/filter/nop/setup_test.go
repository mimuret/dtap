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
package nop_test

import (
	_ "embed"

	"github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/filter/nop"
	"github.com/mimuret/dtap/v2/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nop", func() {
	Context("Setup", func() {
		var (
			err error
			fp  types.FilterPlugin
		)
		BeforeEach(func() {
			fp, err = nop.Setup(json.RawMessage(`{"Name": "nop"}`))
		})
		It("returns nop", func() {
			Expect(err).To(Succeed())
			Expect(fp).To(Equal(&nop.Nop{PluginCommon: plugin.PluginCommon{Name: "nop"}}))
		})
	})
	Context("Filter", func() {
		var (
			fp  *nop.Nop
			dm1 *types.DnstapMessage
			dm2 *types.DnstapMessage
		)
		BeforeEach(func() {
			fp = &nop.Nop{}
			dm1 = &types.DnstapMessage{}
			dm2 = fp.Filter(dm1)
		})
		It("through", func() {
			Expect(dm2).To(Equal(dm1))
		})
	})
})
