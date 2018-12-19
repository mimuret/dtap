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
	"sync"
	"syscall"

	"github.com/mimuret/dtap"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

var (
	flagConfigFile     = flag.String("c", "dtap.toml", "config file path")
	flagLogLevel       = flag.String("d", "info", "log level(debug,info,warn,error,fatal)")
	flagExporterListen = flag.String("e", ":9520", "prometheus exporter listen address")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTION]...\n", os.Args[0])
	flag.PrintDefaults()
}

func outputLoop(outputs []dtap.Output, irbuf *dtap.RBuf) {
	log.Info("start outputLoop")
	for frame := range irbuf.Read() {
		for _, o := range outputs {
			o.SetMessage(frame)
		}
	}
	log.Info("finish outputLoop")
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
	go dtap.PrometheusExporter(context.Background(), *flagExporterListen)
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
		o := dtap.NewDnstapFluentFullOutput(oc)
		output = append(output, o)
	}

	if len(output) == 0 {
		log.Fatal("No output settings")
	}

	iRBuf := dtap.NewRbuf(config.InputMsgBuffer, dtap.TotalRecvInputFrame, dtap.TotalLostInputFrame)
	fatalCh := make(chan error)

	outputCtx, outputCancel := context.WithCancel(context.Background())
	owg := &sync.WaitGroup{}
	for _, o := range output {
		child, _ := context.WithCancel(outputCtx)
		go func() {
			owg.Add(1)
			o.Run(child)
			owg.Done()
		}()
	}

	go outputLoop(output, iRBuf)

	inputCtx, intputCancel := context.WithCancel(context.Background())

	iwg := &sync.WaitGroup{}
	for _, i := range input {
		child, _ := context.WithCancel(inputCtx)
		go func() {
			iwg.Add(1)
			err := i.Run(child, iRBuf)
			iwg.Done()
			if err != nil {
				fatalCh <- err
			}
		}()
	}
	inputFinish := make(chan struct{})
	go func() {
		iwg.Wait()
		close(inputFinish)
	}()

	log.Info("finish boot dtap")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)

	select {
	case <-sigCh:
		log.Info("recieve signal")
	case err := <-fatalCh:
		log.Error(err)
	case <-inputFinish:
		log.Debug("finish all input task")
	}

	log.Info("wait finish input task")
	intputCancel()
	iwg.Wait()
	log.Info("done")

	log.Info("wait finish output task")
	outputCancel()
	owg.Wait()
	log.Info("done")

	iRBuf.Close()

	os.Exit(0)
}
