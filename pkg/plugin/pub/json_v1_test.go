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
package pub_test

import (
	"github.com/mimuret/dtap/v2/pkg/plugin/pub"
	"github.com/mimuret/dtap/v2/pkg/testtool"
	"github.com/mimuret/dtap/v2/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type TestPublisherHandler struct {
	ErrPublish error
	num        int
}

func (t *TestPublisherHandler) Publish(_ []byte) error {
	t.num++
	return t.ErrPublish
}

var _ = Describe("pub", func() {
	Context("JsonV1Publisher", func() {
		var (
			th  *TestPublisherHandler
			op  pub.Publisher
			err error
		)
		BeforeEach(func() {
			th = &TestPublisherHandler{}
			op = pub.NewJsonV1Publisher(1024*1024, th)
		})
		When("NewJsonV1Publisher", func() {
			It("returns JsonV1Publisher", func() {
				Expect(op).NotTo(BeNil())
			})
		})
		Context("Write", func() {
			var (
				dm *types.DnstapMessage
			)
			When("valid message", func() {
				BeforeEach(func() {
					dm = testtool.CreateValidDnstapMessage()
				})
				When("Format is json/v1", func() {
					When("write messages ", func() {
						BeforeEach(func() {
							data, err := dm.ConvertV1JSON()
							Expect(err).To(Succeed())
							maxMsg := (1024*1024 - 2) / (len(data) + 1)
							for i := 0; i < maxMsg; i++ {
								err = op.Write(dm)
								Expect(err).To(Succeed())
							}
						})
						It("succeed", func() {
							Expect(err).To(Succeed())
							Expect(th.num).To(Equal(0))
						})
					})
					When("1 publish", func() {
						BeforeEach(func() {
							data, err := dm.ConvertV1JSON()
							Expect(err).To(Succeed())
							maxMsg := (1024*1024-2)/(len(data)+1) + 1
							for i := 0; i < maxMsg; i++ {
								err = op.Write(dm)
								Expect(err).To(Succeed())
							}
						})
						It("succeed", func() {
							Expect(err).To(Succeed())
							Expect(th.num).To(Equal(1))
						})
					})
					When("2 publish", func() {
						BeforeEach(func() {
							data, err := dm.ConvertV1JSON()
							Expect(err).To(Succeed())
							maxMsg := (1024*1024-2)/(len(data)+1)*2 + 2
							for i := 0; i < maxMsg; i++ {
								err = op.Write(dm)
								Expect(err).To(Succeed())
							}
						})
						It("succeed", func() {
							Expect(err).To(Succeed())
							Expect(th.num).To(Equal(2))
						})
					})
				})
			})
			When("invalid message", func() {
				BeforeEach(func() {
					dm = &types.DnstapMessage{}
				})
				When("Format is json/v1", func() {
					BeforeEach(func() {
						err = op.Write(dm)
					})
					It("returns error", func() {
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(MatchRegexp("failed to convert json"))
					})
				})
			})
		})
	})
})
