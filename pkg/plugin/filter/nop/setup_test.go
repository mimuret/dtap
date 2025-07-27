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
	"context"

	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin/filter/nop"
	"github.com/mimuret/dtap/v3/pkg/testtool"
	"github.com/mimuret/dtap/v3/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nop", func() {
	var (
		ctx context.Context
		msg *types.DnstapMessage
	)

	BeforeEach(func() {
		ctx = context.Background()
		msg = &types.DnstapMessage{
			Labels: map[string]string{
				"example_label": "example_value",
			},
		}
	})

	Context("Setup", func() {
		It("should successfully setup the Nop plugin", func() {
			block := &config.FilterBlock{
				Type: "nop",
				Name: "test_nop",
			}

			plugin, err := nop.Setup(block)
			Expect(err).To(Succeed())
			Expect(plugin).ToNot(BeNil())
		})
	})

	Context("Filter", func() {
		var (
			plugin types.FilterPlugin
		)

		BeforeEach(func() {
			var err error
			plugin, err = nop.Setup(testtool.MustFilterBlock("nop", "test_nop", ``))
			Expect(err).To(Succeed())
		})

		It("should return the same message without modification", func() {
			result := plugin.Filter(ctx, msg)
			Expect(result).To(Equal(msg))
		})
	})
})
