package oteltrace_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	oteltrace "github.com/mimuret/dtap/v2/pkg/plugin/output/otel-trace"
	"github.com/mimuret/dtap/v2/pkg/types"
)

var _ = Describe("output/otel-trace", func() {
	Context("Setup", func() {
		var (
			op  types.OutputPlugin
			err error
		)
		When("invalid config", func() {
			When("type mismatch", func() {
				BeforeEach(func() {
					op, err = oteltrace.Setup(json.RawMessage(`{"ServiceName": 0}`))
				})
				It("returns error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("failed to decode config"))
				})
			})
			When("OTLP.Endpoint is an empty", func() {
				BeforeEach(func() {
					op, err = oteltrace.Setup(json.RawMessage(`{"Name": "otel-trace", "ID":"name-1", "SampleRate": 0.1, "ResourceAttributes": {"A":"a","B":"b"},"OTLP":{}}`))
				})
				It("returns error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("OTLP config error"))
				})
			})
			When("OTLPHTTP.Endpoint is an empty", func() {
				BeforeEach(func() {
					op, err = oteltrace.Setup(json.RawMessage(`{"Name": "otel-trace", "ID":"name-1", "SampleRate": 0.1, "ResourceAttributes": {"A":"a","B":"b"},"OTLPHTTP":{}}`))
				})
				It("returns error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("OTLPHTTP config error"))
				})
			})
		})
		When("vaild", func() {
			BeforeEach(func() {
				op, err = oteltrace.Setup(json.RawMessage(`{"Name": "otel-trace", "ID":"name-1", "SampleRate": 0.1, "ResourceAttributes": {"A":"a","B":"b"},"OTLP": {"ENDPOINT":"localhost:4317"}}`))
			})
			It("returns error", func() {
				Expect(err).To(Succeed())
				Expect(op).NotTo(BeNil())
			})
		})
	})
})
