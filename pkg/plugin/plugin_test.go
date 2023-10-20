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

package plugin_test

import (
	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/testtool"
	"github.com/mimuret/dtap/v2/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type testFilter struct {
	filter bool
}

func (t *testFilter) GetName() string {
	return "test"
}
func (t *testFilter) GetID() string {
	return "test"
}
func (t *testFilter) Filter(dm *types.DnstapMessage) *types.DnstapMessage {
	if t.filter {
		return nil
	}
	return dm
}

var _ = Describe("Plugin", func() {
	Context("Filter", func() {
		var (
			f1, f2 *testFilter
			dm     *types.DnstapMessage
		)
		BeforeEach(func() {
			f1 = &testFilter{}
			f2 = &testFilter{filter: true}
		})
		When("Filtered", func() {
			BeforeEach(func() {
				dm = plugin.FilterPlugins{f1, f2}.Filter(testtool.CreateValidDnstapMessage())
			})
			It("returns nil", func() {
				Expect(dm).To(BeNil())
			})
		})
		When("Pass", func() {
			BeforeEach(func() {
				dm = plugin.FilterPlugins{f1, f1}.Filter(testtool.CreateValidDnstapMessage())
			})
			It("returns nil", func() {
				Expect(dm).NotTo(BeNil())
			})
		})
	})
	Context("PluginCommon", func() {
		var (
			pc plugin.PluginCommon
		)
		BeforeEach(func() {
			pc.Name = "hoge"
		})
		Context("GetName", func() {
			It("returns Name", func() {
				Expect(pc.GetName()).To(Equal("hoge"))
			})
		})
	})
})
