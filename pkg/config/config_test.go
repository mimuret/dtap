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

	"github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/config"
	"github.com/mimuret/dtap/v2/pkg/plugin"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/filter/matcher"
	_ "github.com/mimuret/dtap/v2/pkg/plugin/input/file"
	"github.com/mimuret/dtap/v2/pkg/plugin/output/nop"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	. "github.com/onsi/ginkgo/v2"
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
				ip, err := registry.CreateInputPlugin("file", json.RawMessage(`{"Name": "file","ID":"input_file_1","Path":"/var/tmp/hoge"}`))
				Expect(err).To(Succeed())
				filter1, err := registry.CreateFilterPlugin("matcher", json.RawMessage(`{"Name": "matcher","ID":"filter_matcher_1","Rule": {
					"Op": "AND",
					"Matchers": [
						{
							"Type": "DNS",
							"Name": "qname",
							"Arg": "www.example.jp"
						}
					]
				}}`))
				Expect(err).To(Succeed())
				filter2, err := registry.CreateFilterPlugin("matcher", json.RawMessage(`{"Name": "matcher","ID":"og1_filter_matcher_1","Rule": {
					"Op": "AND",
					"Matchers": [
						{
							"Type": "DNS",
							"Name": "qtype",
							"Arg": "A"
						}
					]
				}}`))
				Expect(err).To(Succeed())
				c := &config.Config{
					InputFilterWorkerNum: 10,
					LogLevel:             "trace",
					ManageHTTPSServer:    ":19520",
					InputBufferConfig: &config.BufferConfig{
						Name: "input_common",
						Size: 100,
					},
					Inputs:  plugin.InputPlugins{ip},
					Filters: plugin.FilterPlugins{filter1},
					OutputGroups: []config.OutputGroupConfig{
						{
							Name: "output-group-0",
							BufferConfig: &config.BufferConfig{
								Name: "output-group-0",
								Size: 10000,
							},
							Filters: plugin.FilterPlugins{filter2},
							Outputs: plugin.OutputPlugins{&nop.NOP{}},
						},
					},
				}
				Expect(cfg.InputBufferConfig).To(Equal(c.InputBufferConfig))
				Expect(cfg.InputFilterWorkerNum).To(Equal(c.InputFilterWorkerNum))
				Expect(cfg.LogLevel).To(Equal(c.LogLevel))
				Expect(cfg.ManageHTTPSServer).To(Equal(c.ManageHTTPSServer))
				Expect(cfg.InputBufferConfig).To(Equal(c.InputBufferConfig))
				Expect(len(cfg.Filters)).To(Equal(len(c.Filters)))
				Expect(len(cfg.Inputs)).To(Equal(len(c.Inputs)))
				Expect(len(cfg.OutputGroups)).To(Equal(len(c.OutputGroups)))
				for i := range cfg.OutputGroups {
					Expect(len(cfg.OutputGroups[i].Filters)).To(Equal(len(c.OutputGroups[i].Filters)))
					Expect(len(cfg.OutputGroups[i].Outputs)).To(Equal(len(c.OutputGroups[i].Filters)))
				}
			})
		})
	})
})
