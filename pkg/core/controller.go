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
	"os/signal"
	"sync"
	"syscall"

	"github.com/mimuret/dtap/v2/pkg/config"
	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

var (
	TotalGlobalFilteredFrame = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dtap_global_filtered_total",
		Help: "The total number of global filtered frames",
	})
	TotalOGFilteredFrame = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dtap_output_group_filtered_total",
		Help: "The total number of output group filtered frames",
	}, []string{"og"})
)

type OutputGroup struct {
	name    string
	buffer  types.Buffer
	filters plugin.FilterPlugins
	outputs plugin.OutputPlugins
}

type Controller struct {
	config *config.Config
	logger *zap.Logger

	inputPlugins plugin.InputPlugins
	inputBuffer  types.Buffer

	filterPlugins plugin.FilterPlugins

	outputGroups []OutputGroup

	onStartup  []func() error
	onShutdown []func() error
}

func NewController(cfg *config.Config, logger *zap.Logger) *Controller {
	return &Controller{
		config: cfg,
		logger: logger,
	}
}

// setup Output Plugin
func (c *Controller) SetupOutputGroup() error {
	var outputGroups []OutputGroup
	for i, ogc := range c.config.OutputGroups {
		if len(ogc.Outputs) == 0 {
			return fmt.Errorf("empty output plugin OutputGroup[%d]", i)
		}
		ob, err := NewOutputBufferFromBufferConfig(ogc.BufferConfig)
		if err != nil {
			return errors.Wrap(err, "failed to create output buffer")
		}
		outputGroups = append(outputGroups, OutputGroup{
			name:    ogc.Name,
			filters: ogc.Filters,
			outputs: ogc.Outputs,
			buffer:  ob,
		})
	}
	c.outputGroups = outputGroups
	return nil
}

// setup controller by config
func (c *Controller) Setup() error {
	// setup plugins
	inputBuf, err := NewBufferFromBufferConfig(c.config.InputBufferConfig, TotalRecvInputFrame, TotalLostInputFrame)
	if err != nil {
		return errors.Wrap(err, "faield to create input buffer")
	}
	c.inputBuffer = inputBuf
	c.inputPlugins = c.config.Inputs
	c.filterPlugins = c.config.Filters
	if err := c.SetupOutputGroup(); err != nil {
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

// add startup function for plugin startup process
func (c *Controller) OnStartup(f func() error) {
	c.onStartup = append(c.onStartup, f)
}

// add shutdown function for plugin shutdown process
func (c *Controller) OnShutdown(f func() error) {
	c.onShutdown = append(c.onShutdown, f)
}

func (c *Controller) PrometheusListen(ctx context.Context) {
	http.Handle("/metrics", promhttp.Handler())
	c.logger.Info("Listening on", zap.String("address", c.config.MetricsListen))
	err := http.ListenAndServe(c.config.MetricsListen, nil)
	if err != nil {
		c.logger.Fatal("failed to listen metrics port", zap.Error(err))
	}
}

// main running function
func (c *Controller) Run(ctx context.Context) error {
	errCh := make(chan error, 128)

	// start inputPlugin
	iwg := sync.WaitGroup{}
	iCtx, iCancel := context.WithCancel(ctx)
	for i, inputPlugin := range c.inputPlugins {
		iwg.Add(1)
		ic := &types.InputContext{
			No:     i,
			Logger: c.logger.With(zap.String("name", inputPlugin.GetName()), zap.Int("no", i)),
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
		for i, outputPlugin := range og.outputs {
			owg.Add(1)
			oc := &types.OutputContext{
				OutputGroup: og.name,
				No:          i,
				Logger:      c.logger.With(zap.String("og", og.name), zap.String("name", outputPlugin.GetName()), zap.Int("no", i)),
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

	// execute startup function
	for _, f := range c.onStartup {
		if err := f(); err != nil {
			c.logger.Error("failed to onStartup", zap.Error(err))
			return err
		}
	}

	defer func() {
		// finish func
		for _, f := range c.onShutdown {
			if err := f(); err != nil {
				c.logger.Error("failed to onShutdown", zap.Error(err))
			}
		}
	}()

	// start main loop
	c.logger.Info("semaphore", zap.Uint("num-worker", c.config.InputFilterWorkerNum))
	iFilterSemaphore := semaphore.NewWeighted(int64(c.config.InputFilterWorkerNum))
	c.logger.Info("Start main loop")
	mainCtx, cancelFunc := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
LOOP:
	for {
		select {
		case <-mainCtx.Done():
			break LOOP
		case err := <-errCh:
			c.logger.Error("plugin error", zap.Error(err))
			cancelFunc()
			break LOOP
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
					TotalGlobalFilteredFrame.Inc()
					return
				}
				for _, og := range c.outputGroups {
					ogdm := dm.DeepCopy()
					// output filter
					ogdm = og.filters.Filter(ogdm)
					if ogdm == nil {
						TotalOGFilteredFrame.WithLabelValues(og.name).Inc()
						continue
					}
					og.buffer.Write(ogdm)
				}
			}()
		}
	}
	return nil
}
