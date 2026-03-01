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
	"context"

	"github.com/mimuret/dtap/v3/pkg/plugin/filter/static"
	"github.com/mimuret/dtap/v3/pkg/testtool"
	"github.com/mimuret/dtap/v3/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Static", func() {
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
		It("should successfully setup with valid HCL", func() {
			plugin, err := static.Setup(testtool.MustFilterBlock("static", "test_static", `deny = true`))
			Expect(err).To(Succeed())
			Expect(plugin).ToNot(BeNil())
		})

		It("should return an error for invalid HCL", func() {
			plugin, err := static.Setup(testtool.MustFilterBlock("static", "test_static", `deny = "invalid_value"`))
			Expect(err).To(HaveOccurred())
			Expect(plugin).To(BeNil())
		})
	})

	Context("Filter", func() {
		var (
			err    error
			plugin types.FilterPlugin
		)

		BeforeEach(func() {
			plugin, err = static.Setup(testtool.MustFilterBlock("static", "test_static", `deny = true`))
			Expect(err).To(Succeed())
		})

		It("should deny the message when deny is true", func() {
			result := plugin.Filter(ctx, msg)
			Expect(result).To(BeNil())
		})

		It("should allow the message when deny is false", func() {
			// Reconfigure the plugin with deny = false
			plugin, err := static.Setup(testtool.MustFilterBlock("static", "test_static", `deny = false`))
			Expect(err).To(Succeed())

			result := plugin.Filter(ctx, msg)
			Expect(result).To(Equal(msg))
		})
	})
})
