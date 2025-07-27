package static

import (
	"context"
	"errors"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/mimuret/dtap/v3/pkg/types"
)

const PLUGIN_NAME = "static"

func init() {
	_ = registry.RegisterFilterPlugin(PLUGIN_NAME, Setup)
}

func Setup(cfg *config.FilterBlock) (types.FilterPlugin, error) {
	fp := &Static{
		FilterBlock: *cfg,
		Deny:        false, // Default value for Deny is false
	}
	// Decode the HCL body into the Relabel struct.
	diags := gohcl.DecodeBody(cfg.Body, nil, fp)
	if diags.HasErrors() {
		// Return an error if there are issues decoding the HCL body.
		return nil, plugin.PluginError(fp, "failed to setup Relabel plugin: %w", errors.Join(diags.Errs()...))
	}

	return fp, nil
}

var _ types.FilterPlugin = &Static{}

// Static plugin for debug
// Example HCL configuration:
//
//	filter "static" "default" {
//	  deny = true
//	}
type Static struct {
	config.FilterBlock
	Deny bool `hcl:"deny,optional"`
}

func (f *Static) Filter(ctx context.Context, msg *types.DnstapMessage) *types.DnstapMessage {
	if f.Deny {
		return nil
	}
	return msg
}
