package sample

import (
	"context"
	"errors"
	"math/rand/v2"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/types"
)

const PLUGIN_NAME = "sample"

func Setup(cfg *config.FilterBlock) (types.FilterPlugin, error) {
	p := &Sample{
		FilterBlock: *cfg,
	}
	// Decode the HCL body into the Relabel struct.
	diags := gohcl.DecodeBody(cfg.Body, nil, p)
	if diags.HasErrors() {
		// Return an error if there are issues decoding the HCL body.
		return nil, plugin.PluginError(p, "failed to setup Relabel plugin: %w", errors.Join(diags.Errs()...))
	}
	if p.SamplingRate <= 0 || p.SamplingRate > 1 {
		return nil, plugin.PluginError(p, "invalid sampling_rate: %f", p.SamplingRate)
	}

	return p, nil
}

type Sample struct {
	config.FilterBlock
	SamplingRate float64 `hcl:"sampling_rate"`
}

func (p *Sample) Filter(ctx context.Context, msg *types.DnstapMessage) *types.DnstapMessage {
	if rand.Float64() <= p.SamplingRate {
		return msg // Keep the message
	}
	return nil
}
