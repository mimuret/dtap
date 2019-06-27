package main

import (
	"context"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	TotalRecvInputFrame = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dtap_input_recv_frame_total",
		Help: "The total number of input frames",
	})
	TotalLostInputFrame = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dtap_input_lost_frame_total",
		Help: "The total number of lost input frames from buffer",
	})
	TotalRecvOutputFrame = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dtap_output_recv_frame_total",
		Help: "The total number of output frames",
	})
	TotalLostOutputFrame = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dtap_output_lost_frame_total",
		Help: "The total number of lost output frames from buffer",
	})
)

func prometheusExporter(ctx context.Context, listen string) {
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(listen, nil)
	if err != nil {
		log.Error(err)
	}
}
