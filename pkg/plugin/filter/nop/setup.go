package nop

import (
	"context"

	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/mimuret/dtap/v3/pkg/types"
)

const PLUGIN_NAME = "nop"

func init() {
	_ = registry.RegisterFilterPlugin(PLUGIN_NAME, Setup)
}

func Setup(cfg *config.FilterBlock) (types.FilterPlugin, error) {
	return &Nop{
		FilterBlock: *cfg,
	}, nil
}

var _ types.FilterPlugin = &Nop{}

// No operation
// Example HCL configuration:
//
//	filter "nop" "default" {
//	  # No operation, just pass through the message
//	}
type Nop struct {
	config.FilterBlock
}

func (f *Nop) Filter(ctx context.Context, t *types.DnstapMessage) *types.DnstapMessage {
	return t
}
