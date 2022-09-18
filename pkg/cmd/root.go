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
	"github.com/mimuret/dtap/v2/pkg/config"
	"github.com/mimuret/dtap/v2/pkg/core"
	_ "github.com/mimuret/dtap/v2/pkg/core/plugins"
	"github.com/mimuret/dtap/v2/pkg/logger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dtap",
	Short: "dtap - DNSTAP message router",
	Long:  `dtap - DNSTAP message router`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.LoadConfig(afero.NewOsFs(), cfgFile)
		if err != nil {
			return err
		}
		l, err := logger.New(c.LogLevel)
		if err != nil {
			return err
		}

		ctl := core.NewController(c, l)
		if err := ctl.Setup(); err != nil {
			l.Fatal("failed to setup", zap.Error(err))
		}
		go ctl.PrometheusListen(cmd.Context())

		return ctl.Run(cmd.Context())
	},
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
}
