package config_test

import (
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
)

var _ = ginkgo.Describe("LoadConfig", func() {
	var (
		fs afero.Fs
	)

	ginkgo.BeforeEach(func() {
		fs = afero.NewMemMapFs()
	})

	ginkgo.Context("when input, output, and filter blocks are valid", func() {
		ginkgo.BeforeEach(func() {
			hclContent := `
input "unix" "unbound" {
  path = "/var/run/unbound/dnstap.sock"
  forward_to = ["filter.mask.default"]
}

filter "mask" "default" {
  MaskLen4 = 24
  MaskLen6 = 56
  forward_to = ["output.stdout.default"]
}

output "stdout" "default" {
  format = "json/v2"
}
`
			err := afero.WriteFile(fs, "/config.hcl", []byte(hclContent), 0644)
			Expect(err).To(Succeed())
		})

		ginkgo.It("should load the configuration successfully", func() {
			cfg, err := config.LoadConfig(fs, "/config.hcl")
			Expect(err).To(Succeed())
			Expect(cfg).ToNot(BeNil())
			Expect(cfg.InputBlocks).To(HaveLen(1))
			Expect(cfg.FilterBlocks).To(HaveLen(1))
			Expect(cfg.OutputBlocks).To(HaveLen(1))
		})
	})

	ginkgo.Context("when duplicate input blocks exist", func() {
		ginkgo.BeforeEach(func() {
			hclContent := `
input "unix" "unbound" {
  path = "/var/run/unbound/dnstap.sock"
  forward_to = ["output.stdout.default"]
}

input "unix" "unbound" {
  path = "/var/run/unbound/dnstap.sock"
  forward_to = ["output.stdout.default"]
}
output "stdout" "default" {}
`
			err := afero.WriteFile(fs, "/config.hcl", []byte(hclContent), 0644)
			Expect(err).To(Succeed())
		})

		ginkgo.It("should return an error for duplicate input blocks", func() {
			_, err := config.LoadConfig(fs, "/config.hcl")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate config block found: input.unix.unbound"))
		})
	})

	ginkgo.Context("when duplicate filter blocks exist", func() {
		ginkgo.BeforeEach(func() {
			hclContent := `
input "unix" "unbound" {
  path = "/var/run/unbound/dnstap.sock"
  forward_to = ["filter.mask.default"]
}
filter "mask" "default" {
  MaskLen4 = 24
  MaskLen6 = 56
  forward_to = ["output.stdout.default"]
}

filter "mask" "default" {
  MaskLen4 = 24
  MaskLen6 = 56
  forward_to = ["output.stdout.default"]
}
output "stdout" "default" {}
`
			err := afero.WriteFile(fs, "/config.hcl", []byte(hclContent), 0644)
			Expect(err).To(Succeed())
		})

		ginkgo.It("should return an error for duplicate filter blocks", func() {
			_, err := config.LoadConfig(fs, "/config.hcl")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate config block found: filter.mask.default"))
		})
	})

	ginkgo.Context("when duplicate output blocks exist", func() {
		ginkgo.BeforeEach(func() {
			hclContent := `
input "unix" "unbound" {
  path = "/var/run/unbound/dnstap.sock"
  forward_to = ["output.stdout.default"]
}
output "stdout" "default" {
  format = "json/v2"
}

output "stdout" "default" {
  format = "json/v2"
}
`
			err := afero.WriteFile(fs, "/config.hcl", []byte(hclContent), 0644)
			Expect(err).To(Succeed())
		})

		ginkgo.It("should return an error for duplicate output blocks", func() {
			_, err := config.LoadConfig(fs, "/config.hcl")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate config block found: output.stdout.default"))
		})
	})

	ginkgo.Context("when Type or Name is missing", func() {
		ginkgo.BeforeEach(func() {
			hclContent := `
input "unix" {
  path = "/var/run/unbound/dnstap.sock"
}

filter "mask" {
  MaskLen4 = 24
  MaskLen6 = 56
}

output "stdout" {
  format = "json/v2"
}
`
			err := afero.WriteFile(fs, "/config.hcl", []byte(hclContent), 0644)
			Expect(err).To(Succeed())
		})

		ginkgo.It("should return an error for missing Type or Name", func() {
			_, err := config.LoadConfig(fs, "/config.hcl")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Missing name for input"))
			Expect(err.Error()).To(ContainSubstring("Missing name for filter"))
			Expect(err.Error()).To(ContainSubstring("Missing name for output;"))
		})
	})
})

var _ = ginkgo.Describe("checkForwardTo", func() {
	var (
		fs afero.Fs
	)

	ginkgo.BeforeEach(func() {
		fs = afero.NewMemMapFs()
	})

	ginkgo.Context("when forward_to is valid", func() {
		ginkgo.BeforeEach(func() {
			hclContent := `
input "unix" "unbound" {
  forward_to = ["filter.mask.default"]
}

filter "mask" "default" {
  forward_to = ["output.stdout.default"]
}

output "stdout" "default" {}
`
			err := afero.WriteFile(fs, "/config.hcl", []byte(hclContent), 0644)
			Expect(err).To(Succeed())
		})

		ginkgo.It("should pass without errors", func() {
			cfg, err := config.LoadConfig(fs, "/config.hcl")
			Expect(err).To(Succeed())
			Expect(cfg).ToNot(BeNil())
		})
	})

	ginkgo.Context("when forward_to references a non-existent block", func() {
		ginkgo.BeforeEach(func() {
			hclContent := `
input "unix" "unbound" {
  forward_to = ["filter.nonexistent.default"]
}

filter "mask" "default" {
  forward_to = ["output.stdout.default"]
}

output "stdout" "default" {}
`
			err := afero.WriteFile(fs, "/config.hcl", []byte(hclContent), 0644)
			Expect(err).To(Succeed())
		})

		ginkgo.It("should return an error for non-existent forward_to", func() {
			_, err := config.LoadConfig(fs, "/config.hcl")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("forward_to `filter.nonexistent.default` does not exist"))
		})
	})

	ginkgo.Context("when forward_to creates a loop", func() {
		ginkgo.BeforeEach(func() {
			hclContent := `
filter "a" "default" {
  forward_to = ["filter.b.default"]
}

filter "b" "default" {
  forward_to = ["filter.c.default"]
}

filter "c" "default" {
  forward_to = ["filter.a.default"]
}
`
			err := afero.WriteFile(fs, "/config.hcl", []byte(hclContent), 0644)
			Expect(err).To(Succeed())
		})

		ginkgo.It("should return an error for a loop in forward_to", func() {
			_, err := config.LoadConfig(fs, "/config.hcl")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("loop detected in forward_to chain"))
		})
	})

	ginkgo.Context("when forward_to is empty", func() {
		ginkgo.BeforeEach(func() {
			hclContent := `
input "unix" "unbound" {
  forward_to = []
}

filter "mask" "default" {
  forward_to = []
}

output "stdout" "default" {}
`
			err := afero.WriteFile(fs, "/config.hcl", []byte(hclContent), 0644)
			Expect(err).To(Succeed())
		})

		ginkgo.It("should return an error for empty forward_to", func() {
			_, err := config.LoadConfig(fs, "/config.hcl")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("forward_to must not be empty"))
		})
	})

	ginkgo.Context("when input block references another input block", func() {
		ginkgo.BeforeEach(func() {
			hclContent := `
input "unix" "unbound" {
  forward_to = ["input.other.default"]
}

input "other" "default" {
  forward_to = ["filter.mask.default"]
}

filter "mask" "default" {
  forward_to = ["output.stdout.default"]
}

output "stdout" "default" {}
`
			err := afero.WriteFile(fs, "/config.hcl", []byte(hclContent), 0644)
			Expect(err).To(Succeed())
		})

		ginkgo.It("should return an error for input block referencing another input block", func() {
			_, err := config.LoadConfig(fs, "/config.hcl")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("input.unix.unbound forward_to `input.other.default` is not supported"))
		})
	})
})
