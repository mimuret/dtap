/*
 * Copyright (c) 2022 Manabu Sonoda
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package core

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/mimuret/dtap/v2/pkg/config"
	"github.com/mimuret/dtap/v2/pkg/logger"
	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/promauto"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/afero"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

type OutputGroup struct {
	name    string
	buffer  types.Buffer
	filters plugin.FilterPlugins
	outputs plugin.OutputPlugins

	filterdCounter prometheus.Counter
}

type controller struct {
	config *config.Config
	logger *zap.Logger

	inputPlugins plugin.InputPlugins
	inputBuffer  types.Buffer

	filterPlugins plugin.FilterPlugins

	filterdCounter prometheus.Counter
	registery      *prometheus.Registry

	outputGroups []OutputGroup

	reloadCh chan struct{}
}

func newController(cfg *config.Config, logger *zap.Logger, registery *prometheus.Registry, reloadCh chan struct{}) *controller {
	return &controller{
		config:    cfg,
		logger:    logger,
		registery: registery,
		reloadCh:  reloadCh,
		filterdCounter: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "dtap",
			Subsystem: "global",
			Name:      "filtered_total",
			Help:      "The total number of global filtered frames",
		}),
	}
}

// setup Output Plugin
func (c *controller) setupOutputGroup() error {
	var outputGroups []OutputGroup
	for i, ogc := range c.config.OutputGroups {
		if len(ogc.Outputs) == 0 {
			return fmt.Errorf("empty output plugin OutputGroup[%d]", i)
		}
		ob, err := NewBufferFromBufferConfig(ogc.BufferConfig,
			promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace:   "dtap",
					Subsystem:   "output",
					Name:        "recv_frame_total",
					Help:        "The total number of output frames",
					ConstLabels: prometheus.Labels{"og": ogc.Name},
				},
			),
			promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace:   "dtap",
					Subsystem:   "output",
					Name:        "lost_frame_total",
					Help:        "The total number of lost output frames from buffer",
					ConstLabels: prometheus.Labels{"og": ogc.Name},
				},
			),
		)
		if err != nil {
			return errors.Wrap(err, "failed to create output buffer")
		}
		outputGroups = append(outputGroups, OutputGroup{
			name:    ogc.Name,
			filters: ogc.Filters,
			outputs: ogc.Outputs,
			buffer:  ob,
			filterdCounter: promauto.NewCounter(prometheus.CounterOpts{
				Namespace:   "dtap",
				Subsystem:   "output",
				Name:        "filtered_total",
				Help:        "The total number of output group filtered frames",
				ConstLabels: prometheus.Labels{"og": ogc.Name},
			}),
		})
	}
	c.outputGroups = outputGroups
	return nil
}

// setup controller by config
func (c *controller) setup() error {
	// setup plugins
	inputBuf, err := NewBufferFromBufferConfig(c.config.InputBufferConfig,
		promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "dtap",
			Subsystem: "input",
			Name:      "recv_frames_total",
			Help:      "The total number of input frames",
		}),
		promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "dtap",
			Subsystem: "input",
			Name:      "lost_frames_total",
			Help:      "The total number of lost input frames from buffer",
		}),
	)
	if err != nil {
		return errors.Wrap(err, "faield to create input buffer")
	}
	c.inputBuffer = inputBuf
	c.inputPlugins = c.config.Inputs
	c.filterPlugins = c.config.Filters
	if err := c.setupOutputGroup(); err != nil {
		return errors.Wrap(err, "failed to create output plugin")
	}
	if len(c.inputPlugins) == 0 {
		return errors.New("Input plugin configuration does not exist")
	}
	if len(c.outputGroups) == 0 {
		return errors.New("Output plugin configuration does not exist")
	}
	return nil
}

func (c *controller) startManageHTTPServer(ctx context.Context) {
	mux := &http.ServeMux{}
	mux.Handle("/metrics", promhttp.InstrumentMetricHandler(
		c.registery, promhttp.HandlerFor(c.registery, promhttp.HandlerOpts{}),
	))
	mux.HandleFunc("/reload", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		c.reloadCh <- struct{}{}
		w.WriteHeader(http.StatusAccepted)
	})
	srv := &http.Server{
		Addr:    c.config.ManageHTTPSServer,
		Handler: mux,
	}
	var errCh = make(chan error)
	c.logger.Info("Start manage http server", zap.String("address", c.config.ManageHTTPSServer))
	go func() {
		errCh <- srv.ListenAndServe()
	}()
	for {
		select {
		case err := <-errCh:
			switch err {
			case http.ErrServerClosed:
				return
			case context.Canceled:
				return
			default:
				c.logger.Fatal("failed to listen metrics port", zap.Error(err))
			}
		case <-ctx.Done():
			_ = srv.Shutdown(ctx)
		}
	}
}

// main running function
func (c *controller) Run(ctx context.Context) error {
	go c.startManageHTTPServer(ctx)
	errCh := make(chan error, 128)

	// start inputPlugin
	iwg := sync.WaitGroup{}
	iCtx, iCancel := context.WithCancel(ctx)
	for _, inputPlugin := range c.inputPlugins {
		iwg.Add(1)
		ic := &types.InputContext{
			Logger: c.logger.With(zap.String("name", inputPlugin.GetName()), zap.String("id", inputPlugin.GetID())),
			Writer: c.inputBuffer,
		}
		go func(ip types.InputPlugin, ic *types.InputContext) {
			ic.Logger.Info("start input plugin")
			err := ip.Start(iCtx, ic)
			ic.Logger.Info("finish input plugin")
			if err != nil {
				errCh <- err
			}
			iwg.Done()
		}(inputPlugin, ic)
	}

	// start outputPlugin
	owg := sync.WaitGroup{}
	oCtx, oCancel := context.WithCancel(ctx)
	for _, og := range c.outputGroups {
		for _, outputPlugin := range og.outputs {
			owg.Add(1)
			oc := &types.OutputContext{
				OutputGroup: og.name,
				Logger:      c.logger.With(zap.String("og", og.name), zap.String("name", outputPlugin.GetName()), zap.String("id", outputPlugin.GetID())),
				Reader:      og.buffer,
			}
			go func(op types.OutputPlugin, oc *types.OutputContext) {
				oc.Logger.Info("start output plugin")
				err := op.Start(oCtx, oc)
				oc.Logger.Info("finish output plugin")
				if err != nil {
					errCh <- err
				}
				owg.Done()
			}(outputPlugin, oc)
		}
	}
	defer func() {
		c.logger.Info("The shutdown process is started.")
		c.logger.Info("Input plugins start the shutdown process.")
		iCancel()
		c.logger.Debug("Waiting for the input shutdown process.")
		iwg.Wait()
		c.logger.Debug("Input plugins shutdown process is completed.")
		c.logger.Info("Output plugin starts the shutdown process")
		oCancel()
		c.logger.Debug("Waiting for the output shutdown process.")
		owg.Wait()
		c.logger.Debug("Output plugins shutdown process is completed.")
		c.logger.Info("Shutdown process is completed.")
	}()

	// start main loop
	c.logger.Info("semaphore", zap.Uint("num-worker", c.config.InputFilterWorkerNum))
	iFilterSemaphore := semaphore.NewWeighted(int64(c.config.InputFilterWorkerNum))
	c.logger.Info("Start main loop")
	mainCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
LOOP:
	for {
		select {
		case <-mainCtx.Done():
			c.logger.Info("cancel recieved")
			break LOOP
		case err := <-errCh:
			c.logger.Error("plugin error", zap.Error(err))
			return fmt.Errorf("plugin error: %w", err)
		// read from input plugin
		case dm := <-c.inputBuffer.Read():
			if dm == nil {
				continue
			}
			// get filter semaphore
			if err := iFilterSemaphore.Acquire(mainCtx, 1); err != nil {
				c.logger.Error("failed to acquire semaphore", zap.Error(err))
				continue
			}
			// input filtering and send outBuffer
			go func() {
				defer func() {
					iFilterSemaphore.Release(1)
				}()
				// input filter
				dm = c.filterPlugins.Filter(dm)
				if dm == nil {
					c.filterdCounter.Inc()
					return
				}
				for _, og := range c.outputGroups {
					ogdm := dm.DeepCopy()
					// output filter
					ogdm = og.filters.Filter(ogdm)
					if ogdm == nil {
						og.filterdCounter.Inc()
						continue
					}
					og.buffer.Write(ogdm)
				}
			}()
		}
	}
	return nil
}

// main running function
func NewRunner(ctx context.Context, cfgFile string, registery *prometheus.Registry, reloadCh chan struct{}) (*controller, *zap.Logger, error) {
	c, err := config.LoadConfig(afero.NewOsFs(), cfgFile)
	if err != nil {
		return nil, nil, err
	}
	l, err := logger.New(c.LogLevel)
	if err != nil {
		return nil, nil, err
	}
	ctl := newController(c, l, registery, reloadCh)
	if err := ctl.setup(); err != nil {
		return nil, nil, fmt.Errorf("failed to setup: %w", err)
	}
	return ctl, l, nil
}
