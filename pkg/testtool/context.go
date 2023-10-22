package testtool

import (
	"github.com/mimuret/dtap/v2/pkg/buffer"
	"github.com/mimuret/dtap/v2/pkg/types"
	"go.uber.org/zap"
)

func NewTestInputContext(w types.Writer) *types.InputContext {
	if w == nil {
		w = buffer.NewRingBuffer(10, nil, nil)
	}
	return &types.InputContext{
		Writer: w,
		Logger: zap.NewExample(),
	}
}

func NewTestOutputContext(r types.Reader) *types.OutputContext {
	if r == nil {
		r = buffer.NewRingBuffer(10, nil, nil)
	}
	return &types.OutputContext{
		OutputGroup: "og-test",
		Reader:      r,
		Logger:      zap.NewExample(),
	}
}
