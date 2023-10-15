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

package cmd

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/mimuret/dtap/v2/pkg/core"
	_ "github.com/mimuret/dtap/v2/pkg/core/plugins"
	"github.com/mimuret/dtap/v2/pkg/promauto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type Runner interface {
	Run(context.Context) error
}

var cfgFile string
var runnerCh chan Runner

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dtap",
	Short: "dtap - DNSTAP message router",
	Long:  `dtap - DNSTAP message router`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			reloadCh   = make(chan struct{}, 1)
			sigHUP     = make(chan os.Signal, 1)
			sigSTOP    = make(chan os.Signal, 1)
			wg         = &sync.WaitGroup{}
			cctx       context.Context
			cancelFunc context.CancelFunc
		)
		signal.Notify(sigHUP, syscall.SIGHUP)
		signal.Notify(sigSTOP, syscall.SIGINT, syscall.SIGTERM)
		defer close(sigHUP)
		defer close(sigSTOP)
		logger, err := makeRunner(cfgFile, reloadCh)
		if err != nil {
			return err
		}

	LOOP:
		for {
			select {
			case r := <-runnerCh:
				wg.Add(1)
				cctx, cancelFunc = context.WithCancel(cmd.Context())
				go func(cctx context.Context) {
					logger.Info("Start runner.")
					if err := r.Run(cctx); err != nil {
						logger.Fatal("runtime error", zap.Error(err))
					}
					wg.Done()
				}(cctx)
			case <-sigSTOP:
				logger.Info("KILL signal recieved.")
				cancelFunc()
				wg.Wait()
				break LOOP
			case <-sigHUP:
				reloadCh <- struct{}{}
			case <-reloadCh:
				if len(runnerCh) != 0 {
					continue
				}
				logger.Info("SIGHUP recieved.")
				newLogger, err := makeRunner(cfgFile, reloadCh)
				if err != nil {
					logger.Error("failed to reload", zap.Error(err))
				} else {
					logger = newLogger
					logger.Info("Stop the current runner to start reloading.")
					cancelFunc()
					wg.Wait()
				}
			}
		}
		return nil
	},
}

func makeRunner(cfgFile string, reloadCh chan struct{}) (*zap.Logger, error) {
	registery := prometheus.NewRegistry()
	promauto.Set(registery)
	runner, l, err := core.NewRunner(context.Background(), cfgFile, registery, reloadCh)
	if err != nil {
		return nil, err
	}
	runnerCh <- runner
	return l, nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file")
	runnerCh = make(chan Runner, 1)
}
