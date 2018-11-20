/*
 * Copyright (c) 2018 Manabu Sonoda
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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/mimuret/dtap"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

var (
	flagConfigFile = flag.String("c", "dtap.toml", "config file path")
	flagLogLevel   = flag.String("d", "info", "log level(debug,info,warn,error,fatal)")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTION]...\n", os.Args[0])
	flag.PrintDefaults()
}

func outputLoop(ctx context.Context, sockets []dtap.Output, inputChannel chan []byte) {
	for {
		select {
		case frame := <-inputChannel:
			for _, o := range sockets {
				o.GetOutputChannel() <- frame
			}
		case <-ctx.Done():
			break
		}
	}
}
func outputError(errCh chan error) {
	for err := range errCh {
		log.Warnf("%+v", err)
	}
}

func fatalCheck(err error) {
	if err != nil {
		log.Fatalf("%+v", err)
	}
}

func main() {
	var err error

	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Usage = usage

	flag.Parse()
	// set log level
	switch *flagLogLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	default:
		usage()
		os.Exit(1)
	}
	var input []dtap.Input
	var output []dtap.Output

	config, err := dtap.NewConfigFromFile(*flagConfigFile)
	fatalCheck(err)
	for _, ic := range config.InputFile {
		i, err := dtap.NewDnstapFstrmFileInput(ic)
		fatalCheck(err)
		input = append(input, i)
	}

	for _, ic := range config.InputTCP {
		i, err := dtap.NewDnstapFstrmTCPSocketInput(ic)
		fatalCheck(err)
		input = append(input, i)
	}

	for _, ic := range config.InputUnix {
		i, err := dtap.NewDnstapFstrmUnixSocketInput(ic)
		fatalCheck(err)
		input = append(input, i)
	}

	if len(input) == 0 {
		log.Fatal("No input settings")
	}

	for _, oc := range config.OutputFile {
		o := dtap.NewDnstapFstrmFileOutput(oc)
		fatalCheck(err)
		output = append(output, o)
	}

	for _, oc := range config.OutputTCP {
		o := dtap.NewDnstapFstrmTCPSocketOutput(oc)
		output = append(output, o)
	}

	for _, oc := range config.OutputUnix {
		o := dtap.NewDnstapFstrmUnixSockOutput(oc)
		output = append(output, o)
	}

	for _, oc := range config.OutputFluent {
		if o, err := dtap.NewDnstapFluentFullOutput(oc); err != nil {
			fatalCheck(err)
		} else {
			output = append(output, o)
		}
	}

	if len(output) == 0 {
		log.Fatal("No output settings")
	}

	inputChannel := make(chan []byte, config.InputChannelSize)
	errCh := make(chan error, config.InputChannelSize)
	go outputError(errCh)
	log.Info("start err outputer")
	ctx, cancel := context.WithCancel(context.Background())
	for _, o := range output {
		go o.Run(ctx, errCh)
	}
	log.Info("start output loop")
	go outputLoop(ctx, output, inputChannel)
	log.Info("start main output loop")
	for _, i := range input {
		go i.Run(ctx, inputChannel, errCh)
	}
	log.Info("start input loop")
	log.Info("finish boot dtap")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	select {
	case <-sigCh:
		cancel()
	}
	for _, o := range output {
		for !o.Finished() {
			time.Sleep(1 * time.Second)
		}
	}
	close(inputChannel)

}
