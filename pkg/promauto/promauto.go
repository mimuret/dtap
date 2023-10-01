package promauto

import (
	"github.com/prometheus/client_golang/prometheus"
	originpromauto "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	resigerer prometheus.Registerer
	factory   originpromauto.Factory
)

func InitRegistry() {
	resigerer = prometheus.NewRegistry()
	factory = originpromauto.With(resigerer)
}

func init() {
	resigerer = prometheus.NewRegistry()
	factory = originpromauto.With(resigerer)
}

var (
	NewCounter      = factory.NewCounter
	NewCounterVec   = factory.NewCounterVec
	NewCounterFunc  = factory.NewCounterFunc
	NewGauge        = factory.NewGauge
	NewGaugeVec     = factory.NewGaugeVec
	NewGaugeFunc    = factory.NewGaugeFunc
	NewSummary      = factory.NewSummary
	NewSummaryVec   = factory.NewSummaryVec
	NewHistogram    = factory.NewHistogram
	NewHistogramVec = factory.NewHistogramVec
	NewUntypedFunc  = factory.NewUntypedFunc
)
