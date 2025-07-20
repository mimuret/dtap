package multi_test

import (
	"context"

	"github.com/goccy/go-json"
	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/output/multi"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// モックプラグインを登録
type MockOutputPlugin struct {
	plugin.PluginCommon
	startCalled bool
}

func (m *MockOutputPlugin) Start(ctx context.Context, oc *types.OutputContext) error {
	m.startCalled = true
	return nil
}

var _ = BeforeSuite(func() {
	_ = registry.RegisterOutputPlugin("mock", func(config json.RawMessage) (types.OutputPlugin, error) {
		o := &MockOutputPlugin{}
		if err := json.Unmarshal(config, o); err != nil {
			return nil, err
		}
		return o, nil
	})
})

var _ = Describe("MultiRunner", func() {
	Context("Setup", func() {
		It("should succeed with valid configuration", func() {
			config := `{
                "Name": "multi",
                "ID": "test-multi",
                "Concurency": 2,
                "Plugin": {
                    "Name": "mock"
                }
            }`

			plugin, err := multi.Setup([]byte(config))
			Expect(err).To(Succeed())
			multiRunner, ok := plugin.(*multi.MultiRunner)
			Expect(ok).To(BeTrue())
			Expect(multiRunner.Concurency).To(Equal(uint(2)))
			Expect(multiRunner.Plugins()).To(HaveLen(2))
			Expect(multiRunner.Plugins()[0].GetName()).To(Equal("mock"))
			Expect(multiRunner.Plugins()[0].GetID()).To(Equal("test-multi-0"))
			Expect(multiRunner.Plugins()[1].GetName()).To(Equal("mock"))
			Expect(multiRunner.Plugins()[1].GetID()).To(Equal("test-multi-1"))
		})

		It("should fail when Concurency is 0", func() {
			config := `{
                "Name": "multi",
                "ID": "test-multi",
                "Concurency": 0,
                "Plugin": {
                    "Name": "mock",
                    "ID": "mock-plugin"
                }
            }`

			_, err := multi.Setup([]byte(config))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Concurency must be greater than 0"))
		})

		It("should fail when Plugin is invalid", func() {
			config := `{
                "Name": "multi",
                "ID": "test-multi",
                "Concurency": 1,
                "Plugin": {
                    "Name": "invalid-plugin",
                    "ID": "mock-plugin"
                }
            }`

			_, err := multi.Setup([]byte(config))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create output plugin"))
		})
	})

	Context("Start", func() {
		It("should call Start on all plugins", func() {
			config := `{
                "Name": "multi",
                "ID": "test-multi",
                "Concurency": 2,
                "Plugin": {
                    "Name": "mock",
                    "ID": "mock-plugin"
                }
            }`

			plugin, err := multi.Setup([]byte(config))
			Expect(err).To(Succeed())
			multiRunner, ok := plugin.(*multi.MultiRunner)
			Expect(ok).To(BeTrue())

			ctx := context.Background()
			oc := &types.OutputContext{}

			err = multiRunner.Start(ctx, oc)
			Expect(err).To(Succeed())

			for _, p := range multiRunner.Plugins() {
				mockPlugin, ok := p.(*MockOutputPlugin)
				Expect(ok).To(BeTrue())
				Expect(mockPlugin.startCalled).To(BeTrue())
			}
		})
	})
})
