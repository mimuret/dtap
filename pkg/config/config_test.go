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

package config_test

import (
	_ "embed"

	"github.com/mimuret/dtap/v2/pkg/config"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/filter/matcher"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/output/nop"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
)

//go:embed testdata/valid_config.yaml
var validYAMLConfig []byte

//go:embed testdata/valid_config.json
var validJSONConfig []byte

var _ = Describe("config", func() {
	Context("LoadConfig", func() {
		var (
			fs  = afero.NewMemMapFs()
			err error
			cfg *config.Config
		)
		BeforeEach(func() {
			fs = afero.NewMemMapFs()

			f, err := fs.Create("/invalid.cfg")
			Expect(err).To(Succeed())
			_, err = f.Write(validJSONConfig)
			Expect(err).To(Succeed())
			f.Close()

			f, err = fs.Create("/valid-yaml.yaml")
			Expect(err).To(Succeed())
			_, err = f.Write(validYAMLConfig)
			Expect(err).To(Succeed())
			f.Close()

			f, err = fs.Create("/valid-json.json")
			Expect(err).To(Succeed())
			_, err = f.Write(validJSONConfig)
			Expect(err).To(Succeed())
			f.Close()
		})
		When("file not exist", func() {
			BeforeEach(func() {
				cfg, err = config.LoadConfig(fs, "/not-exist.yaml")
			})
			It("returns err", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("failed to open config file"))
			})
		})
		When("file type is not yaml and json", func() {
			BeforeEach(func() {
				cfg, err = config.LoadConfig(fs, "/invalid.cfg")
			})
			It("returns err", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("unsupported config format"))
			})
		})
		When("valid YAML config", func() {
			BeforeEach(func() {
				cfg, err = config.LoadConfig(fs, "/valid-yaml.yaml")
			})
			It("returns Config", func() {
				Expect(err).To(Succeed())
				Expect(cfg).NotTo(BeNil())
			})
		})
	})
})
