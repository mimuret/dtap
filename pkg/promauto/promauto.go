package promauto

import (
	"github.com/prometheus/client_golang/prometheus"
	origin "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	f          *origin.Factory
	registerer prometheus.Registerer
)

func DefaultRegisterer() prometheus.Registerer {
	return registerer
}

func Set(r prometheus.Registerer) {
	registerer = r
	fac := origin.With(r)
	f = &fac
}

func NewCounter(opts prometheus.CounterOpts) prometheus.Counter {
	if f == nil {
		return prometheus.NewCounter(opts)
	}
	return f.NewCounter(opts)
}

// NewCounterVec works like the function of the same name in the prometheus
// package but it automatically registers the CounterVec with the
// prometheus.DefaultRegisterer. If the registration fails, NewCounterVec
// panics.
func NewCounterVec(opts prometheus.CounterOpts, labelNames []string) *prometheus.CounterVec {
	if f == nil {
		return prometheus.NewCounterVec(opts, labelNames)
	}
	return f.NewCounterVec(opts, labelNames)
}

// NewCounterFunc works like the function of the same name in the prometheus
// package but it automatically registers the CounterFunc with the
// prometheus.DefaultRegisterer. If the registration fails, NewCounterFunc
// panics.
func NewCounterFunc(opts prometheus.CounterOpts, function func() float64) prometheus.CounterFunc {
	if f == nil {
		return prometheus.NewCounterFunc(opts, function)
	}
	return f.NewCounterFunc(opts, function)
}

// NewGauge works like the function of the same name in the prometheus package
// but it automatically registers the Gauge with the
// prometheus.DefaultRegisterer. If the registration fails, NewGauge panics.
func NewGauge(opts prometheus.GaugeOpts) prometheus.Gauge {
	if f == nil {
		return prometheus.NewGauge(opts)
	}
	return f.NewGauge(opts)
}

// NewGaugeVec works like the function of the same name in the prometheus
// package but it automatically registers the GaugeVec with the
// prometheus.DefaultRegisterer. If the registration fails, NewGaugeVec panics.
func NewGaugeVec(opts prometheus.GaugeOpts, labelNames []string) *prometheus.GaugeVec {
	if f == nil {
		return prometheus.NewGaugeVec(opts, labelNames)
	}
	return f.NewGaugeVec(opts, labelNames)
}

// NewGaugeFunc works like the function of the same name in the prometheus
// package but it automatically registers the GaugeFunc with the
// prometheus.DefaultRegisterer. If the registration fails, NewGaugeFunc panics.
func NewGaugeFunc(opts prometheus.GaugeOpts, function func() float64) prometheus.GaugeFunc {
	if f == nil {
		return prometheus.NewGaugeFunc(opts, function)
	}
	return f.NewGaugeFunc(opts, function)
}

// NewSummary works like the function of the same name in the prometheus package
// but it automatically registers the Summary with the
// prometheus.DefaultRegisterer. If the registration fails, NewSummary panics.
func NewSummary(opts prometheus.SummaryOpts) prometheus.Summary {
	if f == nil {
		return prometheus.NewSummary(opts)
	}
	return f.NewSummary(opts)
}

// NewSummaryVec works like the function of the same name in the prometheus
// package but it automatically registers the SummaryVec with the
// prometheus.DefaultRegisterer. If the registration fails, NewSummaryVec
// panics.
func NewSummaryVec(opts prometheus.SummaryOpts, labelNames []string) *prometheus.SummaryVec {
	if f == nil {
		return prometheus.NewSummaryVec(opts, labelNames)
	}
	return f.NewSummaryVec(opts, labelNames)
}

// NewHistogram works like the function of the same name in the prometheus
// package but it automatically registers the Histogram with the
// prometheus.DefaultRegisterer. If the registration fails, NewHistogram panics.
func NewHistogram(opts prometheus.HistogramOpts) prometheus.Histogram {
	if f == nil {
		return prometheus.NewHistogram(opts)
	}
	return f.NewHistogram(opts)
}

// NewHistogramVec works like the function of the same name in the prometheus
// package but it automatically registers the HistogramVec with the
// prometheus.DefaultRegisterer. If the registration fails, NewHistogramVec
// panics.
func NewHistogramVec(opts prometheus.HistogramOpts, labelNames []string) *prometheus.HistogramVec {
	if f == nil {
		return prometheus.NewHistogramVec(opts, labelNames)
	}
	return f.NewHistogramVec(opts, labelNames)
}

// NewUntypedFunc works like the function of the same name in the prometheus
// package but it automatically registers the UntypedFunc with the
// prometheus.DefaultRegisterer. If the registration fails, NewUntypedFunc
// panics.
func NewUntypedFunc(opts prometheus.UntypedOpts, function func() float64) prometheus.UntypedFunc {
	if f == nil {
		return prometheus.NewUntypedFunc(opts, function)
	}
	return f.NewUntypedFunc(opts, function)
}
