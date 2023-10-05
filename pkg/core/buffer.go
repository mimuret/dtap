package core

import (
	"github.com/mimuret/dtap/v2/pkg/buffer"
	"github.com/mimuret/dtap/v2/pkg/config"
	"github.com/mimuret/dtap/v2/pkg/types"
)

func NewBufferFromBufferConfig(c *config.BufferConfig, inCounter, lostCounter types.Counter) (types.Buffer, error) {
	return buffer.NewRingBuffer(c.Size, inCounter, lostCounter), nil
}
