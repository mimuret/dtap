package core

import (
	"github.com/mimuret/dtap/v2/pkg/buffer"
	"github.com/mimuret/dtap/v2/pkg/config"
	"github.com/mimuret/dtap/v2/pkg/promauto"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	TotalRecvInputFrame = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "dtap",
		Subsystem: "input",
		Name:      "recv_frame_total",
		Help:      "The total number of input frames",
	})
	TotalLostInputFrame = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "dtap",
		Subsystem: "input",
		Name:      "lost_frame_total",
		Help:      "The total number of lost input frames from buffer",
	})
	TotalRecvOutputFrame = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "dtap",
			Subsystem: "output",
			Name:      "recv_frame_total",
			Help:      "The total number of output frames",
		},
		[]string{"og"},
	)
	TotalLostOutputFrame = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "dtap",
			Subsystem: "output",
			Name:      "lost_frame_total",
			Help:      "The total number of lost output frames from buffer",
		},
		[]string{"og"},
	)
)

func NewBufferFromBufferConfig(c *config.BufferConfig, inCounter, lostCounter types.Counter) (types.Buffer, error) {
	return buffer.NewRingBuffer(c.Size, inCounter, lostCounter), nil
}

func NewOutputBufferFromBufferConfig(c *config.BufferConfig) (types.Buffer, error) {
	var (
		outCounter, lostCounter types.Counter
	)
	if c.GetName() != "" {
		outCounter = TotalRecvOutputFrame.WithLabelValues(c.GetName())
		lostCounter = TotalLostOutputFrame.WithLabelValues(c.GetName())
	}
	return NewBufferFromBufferConfig(c, outCounter, lostCounter)
}
