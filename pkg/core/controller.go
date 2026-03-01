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
	"net/http/pprof"
	"sync"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/logger"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/promauto"
	"github.com/mimuret/dtap/v3/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/afero"
	"go.uber.org/zap"
)

type Options struct {
	ManageHTTPSServer string
	Debug             bool
	LogLevel          string
}

type controller struct {
	config *config.Config
	logger *zap.Logger

	options *Options

	inputRunners  map[string]*plugin.InputRunner
	filterRunners map[string]*plugin.FilterRunner
	outputRunners map[string]*plugin.OutputRunner

	filterdCounter prometheus.Counter

	registery *prometheus.Registry

	reloadCh chan struct{}

	ready bool
}

func newController(cfg *config.Config, logger *zap.Logger, registery *prometheus.Registry, reloadCh chan struct{}, options *Options) *controller {
	return &controller{
		config:    cfg,
		logger:    logger,
		registery: registery,
		reloadCh:  reloadCh,
		options:   options,
		filterdCounter: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "dtap",
			Subsystem: "global",
			Name:      "filtered_total",
			Help:      "The total number of global filtered frames",
		}),
	}
}

func (c *controller) setupInputPlugins() error {
	var err error
	c.inputRunners, err = plugin.SetupInputRunners(c.config.InputBlocks)
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) setupOutputPlugins() error {
	var err error
	c.outputRunners, err = plugin.SetupOutputRunners(c.config.OutputBlocks)
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) setupFilterPlugins() error {
	var err error
	c.filterRunners, err = plugin.SetupFilterRunners(c.config.FilterBlocks)
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) setupForwardTo() error {
	// setup forwardTo for output plugins
	receivers := make(map[string]types.Receiver, len(c.outputRunners)+len(c.filterRunners))
	for name, p := range c.outputRunners {
		receivers[name] = p
	}
	for name, p := range c.filterRunners {
		receivers[name] = p
	}

	// setup forwardTo for input plugins
	for _, block := range c.config.InputBlocks {
		forwardTo := make([]types.Writer, 0, len(block.ForwardToString))
		for _, name := range block.ForwardToString {
			if receiver, ok := receivers[name]; ok {
				forwardTo = append(forwardTo, receiver)
			} else {
				return plugin.PluginError(block, "forward_to %s not found in output or filter plugins", name)
			}
		}
		c.inputRunners[block.GetFullName()].SetupForwardTo(forwardTo)
	}
	// setup forwardTo for filter plugins
	for _, block := range c.config.FilterBlocks {
		forwardTo := make([]types.Writer, 0, len(block.ForwardToString))
		for _, name := range block.ForwardToString {
			if receiver, ok := receivers[name]; ok {
				forwardTo = append(forwardTo, receiver)
			} else {
				return fmt.Errorf("forward_to %s not found in output or filter plugins", name)
			}
		}
		c.filterRunners[block.GetFullName()].SetupForwardTo(forwardTo)
	}
	return nil
}

// setup Output Plugin
// setup controller by config
func (c *controller) setup() error {
	if err := c.setupInputPlugins(); err != nil {
		return err
	}
	if err := c.setupOutputPlugins(); err != nil {
		return err
	}
	if err := c.setupFilterPlugins(); err != nil {
		return err
	}
	if err := c.setupForwardTo(); err != nil {
		return err
	}
	return nil
}

func (c *controller) startManageHTTPServer(ctx context.Context) {
	mux := &http.ServeMux{}
	mux.Handle("/metrics", promhttp.InstrumentMetricHandler(
		c.registery, promhttp.HandlerFor(c.registery, promhttp.HandlerOpts{}),
	))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		if c.ready {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready"))
		}
	})
	mux.HandleFunc("/reload", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		c.reloadCh <- struct{}{}
		w.WriteHeader(http.StatusAccepted)
	})
	if c.options.Debug {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
	srv := &http.Server{
		Addr:    c.options.ManageHTTPSServer,
		Handler: mux,
	}
	var errCh = make(chan error)
	c.logger.Debug("Starting management HTTP server", zap.String("address", c.options.ManageHTTPSServer))
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

func (c *controller) startRunners(ctx context.Context, runners map[string]plugin.Runner, errCh chan error) (context.CancelFunc, *sync.WaitGroup) {
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(ctx)
	for _, p := range runners {
		wg.Add(1)
		rctx := ctxzap.ToContext(ctx, c.logger.With(
			zap.String("plugin", p.GetFullName()),
		))
		go func(r plugin.Runner, ctx context.Context) {
			ctxzap.Debug(ctx, "start plugin")
			r.Start(ctx, errCh)
			ctxzap.Debug(ctx, "finish plugin")
			wg.Done()
		}(p, rctx)
	}
	return cancel, wg
}

// startOutputPlugins starts all output plugins and returns a cancel function and a wait group
func (c *controller) startOutputRunners(ctx context.Context, errCh chan error) (context.CancelFunc, *sync.WaitGroup) {
	runners := make(map[string]plugin.Runner, len(c.outputRunners))
	for name, runner := range c.outputRunners {
		runners[name] = runner
	}
	return c.startRunners(ctx, runners, errCh)
}

// startOutputPlugins starts all output plugins and returns a cancel function and a wait group
func (c *controller) startFilterRunners(ctx context.Context, errCh chan error) (context.CancelFunc, *sync.WaitGroup) {
	runners := make(map[string]plugin.Runner, len(c.filterRunners))
	for name, runner := range c.filterRunners {
		runners[name] = runner
	}
	return c.startRunners(ctx, runners, errCh)
}

// startOutputPlugins starts all output plugins and returns a cancel function and a wait group
func (c *controller) startInputRunners(ctx context.Context, errCh chan error) (context.CancelFunc, *sync.WaitGroup) {
	runners := make(map[string]plugin.Runner, len(c.filterRunners))
	for name, runner := range c.inputRunners {
		runners[name] = runner
	}
	return c.startRunners(ctx, runners, errCh)
}

// main running function
func (c *controller) Run(ctx context.Context) error {
	go c.startManageHTTPServer(ctx)
	filterErrCh := make(chan error, 128)
	inputErrCh := make(chan error, 128)
	outputErrCh := make(chan error, 128)
	fCancel, fwg := c.startFilterRunners(ctx, filterErrCh)
	iCancel, iwg := c.startInputRunners(ctx, inputErrCh)
	oCancel, owg := c.startOutputRunners(ctx, outputErrCh)

	c.ready = true

	defer func() {
		c.logger.Info("Shutdown process has started.")
		c.logger.Debug("Starting shutdown process for input plugins.")
		iCancel()
		c.logger.Debug("Waiting for input plugins to shut down.")
		iwg.Wait()
		close(inputErrCh)
		for err := range inputErrCh {
			c.logger.Error("Plugin error", zap.Error(err))
		}
		c.logger.Debug("Input plugins have been shut down.")

		c.logger.Debug("Starting shutdown process for filter plugins.")
		fCancel()
		c.logger.Debug("Waiting for filter plugins to shut down.")
		fwg.Wait()
		close(filterErrCh)
		for err := range filterErrCh {
			c.logger.Error("Plugin error", zap.Error(err))
		}

		c.logger.Debug("Filter plugins have been shut down.")

		c.logger.Debug("Starting shutdown process for output plugins.")
		oCancel()
		c.logger.Debug("Waiting for output plugins to shut down.")
		owg.Wait()
		close(outputErrCh)
		for err := range outputErrCh {
			c.logger.Error("Plugin error", zap.Error(err))
		}
		c.logger.Debug("Output plugins have been shut down.")

		c.logger.Info("Shutdown process has completed.")
	}()
	mainCtx, cancelFunc := context.WithCancel(ctx)

	// start main loop
	defer cancelFunc()
LOOP:
	for {
		select {
		case <-mainCtx.Done():
			c.ready = false
			c.logger.Info("Cancellation signal received.")
			break LOOP
		case err := <-inputErrCh:
			c.ready = false
			c.logger.Error("Plugin error", zap.Error(err))
			return fmt.Errorf("error occurred in plugin")
		case err := <-filterErrCh:
			c.ready = false
			c.logger.Error("Plugin error", zap.Error(err))
			return fmt.Errorf("error occurred in plugin")
		case err := <-outputErrCh:
			c.ready = false
			c.logger.Error("Plugin error", zap.Error(err))
			return fmt.Errorf("error occurred in plugin")
		}
	}
	return nil
}

// main running function
func NewRunner(cfgFile string, registery *prometheus.Registry, reloadCh chan struct{}, options *Options) (*controller, *zap.Logger, error) {
	c, err := config.LoadConfig(afero.NewOsFs(), cfgFile)
	if err != nil {
		return nil, nil, err
	}
	l, err := logger.New(options.LogLevel)
	if err != nil {
		return nil, nil, err
	}
	ctl := newController(c, l, registery, reloadCh, options)
	if err := ctl.setup(); err != nil {
		return nil, nil, fmt.Errorf("failed to setup: %w", err)
	}
	return ctl, l, nil
}
