package relabel_test

import (
	"context"

	"github.com/mimuret/dtap/v3/pkg/plugin/filter/relabel"
	"github.com/mimuret/dtap/v3/pkg/testtool"
	"github.com/mimuret/dtap/v3/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Relabel", func() {
	var (
		ctx context.Context
		msg *types.DnstapMessage
	)

	BeforeEach(func() {
		ctx = context.Background()
		msg = &types.DnstapMessage{
			Labels: map[string]string{
				"original_label": "original_value",
			},
		}
	})

	Context("Setup", func() {
		It("should successfully setup with valid HCL", func() {
			hclData := `
replace_label "old_label" "new_label" {
	template = "{{ .Labels.original_label }}_modified"
}
drop_label "original_label" {}
`
			plugin, err := relabel.Setup(testtool.MustFilterBlock("relabel", "test_relabel", hclData))
			Expect(err).To(Succeed())
			Expect(plugin).ToNot(BeNil())
		})

		It("should return an error for invalid HCL", func() {
			hclData := `
replace_label "" "" {
	template = "{{ .Labels.original_label }}_modified"
}
`
			plugin, err := relabel.Setup(testtool.MustFilterBlock("relabel", "test_relabel", hclData))
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
			hclData := `
replace_label "new_label" "new_label" {
	template = "{{ .Labels.original_label }}_modified"
}
drop_label  "original_label" {}
`
			plugin, err = relabel.Setup(testtool.MustFilterBlock("relabel", "test_relabel", hclData))
			Expect(err).To(Succeed())
		})

		It("should apply relabeling rules correctly", func() {
			result := plugin.Filter(ctx, msg)

			// Check that the original label was dropped
			Expect(result.Labels).ToNot(HaveKey("original_label"))

			// Check that the new label was added
			Expect(result.Labels).To(HaveKeyWithValue("new_label", "original_value_modified"))
		})
	})
})
