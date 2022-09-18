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

package registry_test

import (
	json "github.com/goccy/go-json"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testInputPluginFunc(json.RawMessage) (types.InputPlugin, error) {
	return nil, nil
}
func testOutputPluginFunc(json.RawMessage) (types.OutputPlugin, error) {
	return nil, nil
}
func testFilterPluginFunc(json.RawMessage) (types.FilterPlugin, error) {
	return nil, nil
}

var _ = Describe("Plugin", func() {
	var (
		err error
	)
	Context("RegisterInputPlugin", func() {
		When("name is an empty", func() {
			BeforeEach(func() {
				err = registry.RegisterInputPlugin("", testInputPluginFunc)
			})
			It("return err", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("name is an empty"))
			})
		})
		When("invalid newFunc", func() {
			BeforeEach(func() {
				err = registry.RegisterInputPlugin("example", nil)
			})
			It("return err", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("setupFunc is nil"))
			})
		})
		When("valid args", func() {
			BeforeEach(func() {
				err = registry.RegisterInputPlugin("example", testInputPluginFunc)
			})
			It("return err", func() {
				Expect(err).To(Succeed())
			})
		})
	})
	Context("RegisterOutputPlugin", func() {
		When("name is an empty", func() {
			BeforeEach(func() {
				err = registry.RegisterOutputPlugin("", testOutputPluginFunc)
			})
			It("return err", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("name is an empty"))
			})
		})
		When("invalid func", func() {
			BeforeEach(func() {
				err = registry.RegisterOutputPlugin("example", nil)
			})
			It("return err", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("setupFunc is nil"))
			})
		})
		When("valid args", func() {
			BeforeEach(func() {
				err = registry.RegisterOutputPlugin("example", testOutputPluginFunc)
			})
			It("return err", func() {
				Expect(err).To(Succeed())
			})
		})
	})
	Context("RegisterFilterPlugin", func() {
		When("name is an empty", func() {
			BeforeEach(func() {
				err = registry.RegisterFilterPlugin("", testFilterPluginFunc)
			})
			It("return err", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("name is an empty"))
			})
		})
		When("invalid func", func() {
			BeforeEach(func() {
				err = registry.RegisterFilterPlugin("example", nil)
			})
			It("return err", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("setupFunc is nil"))
			})
		})
		When("valid args", func() {
			BeforeEach(func() {
				err = registry.RegisterFilterPlugin("example", testFilterPluginFunc)
			})
			It("return err", func() {
				Expect(err).To(Succeed())
			})
		})
	})
})
