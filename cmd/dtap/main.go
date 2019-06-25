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

func outputLoop(output []dtap.Output, irbuf *dtap.RBuf) {
	log.Info("start outputLoop")
	for frame := range irbuf.Read() {
		for _, o := range output {
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
	go prometheusExporter(context.Background(), *flagExporterListen)
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
		params := &dtap.DnstapOutputParams{
			BufferSize:  oc.Buffer.GetBufferSize(),
			InCounter:   TotalRecvOutputFrame,
			LostCounter: TotalLostInputFrame,
		}
		o := dtap.NewDnstapFstrmFileOutput(oc, params)
		output = append(output, o)
	}

	for _, oc := range config.OutputTCP {
		params := &dtap.DnstapOutputParams{
			BufferSize:  oc.Buffer.GetBufferSize(),
			InCounter:   TotalRecvOutputFrame,
			LostCounter: TotalLostInputFrame,
		}
		o := dtap.NewDnstapFstrmTCPSocketOutput(oc, params)
		output = append(output, o)
	}

	for _, oc := range config.OutputUnix {
		params := &dtap.DnstapOutputParams{
			BufferSize:  oc.Buffer.GetBufferSize(),
			InCounter:   TotalRecvOutputFrame,
			LostCounter: TotalLostInputFrame,
		}
		o := dtap.NewDnstapFstrmUnixSockOutput(oc, params)
		output = append(output, o)
	}

	for _, oc := range config.OutputFluent {
		params := &dtap.DnstapOutputParams{
			BufferSize:  oc.Buffer.GetBufferSize(),
			InCounter:   TotalRecvOutputFrame,
			LostCounter: TotalLostInputFrame,
		}
		o := dtap.NewDnstapFluentdOutput(oc, params)
		output = append(output, o)
		if oc.Flat.GetIPHashSaltPath() != "" {
			go oc.Flat.WatchSalt(context.Background())
		}
	}

	for _, oc := range config.OutputKafka {
		params := &dtap.DnstapOutputParams{
			BufferSize:  oc.Buffer.GetBufferSize(),
			InCounter:   TotalRecvOutputFrame,
			LostCounter: TotalLostInputFrame,
		}
		o := dtap.NewDnstapKafkaOutput(oc, params)
		output = append(output, o)
		if oc.Flat.GetIPHashSaltPath() != "" {
			go oc.Flat.WatchSalt(context.Background())
		}
	}

	for _, oc := range config.OutputNats {
		params := &dtap.DnstapOutputParams{
			BufferSize:  oc.Buffer.GetBufferSize(),
			InCounter:   TotalRecvOutputFrame,
			LostCounter: TotalLostInputFrame,
		}
		o := dtap.NewDnstapNatsOutput(oc, params)
		output = append(output, o)
		if oc.Flat.GetIPHashSaltPath() != "" {
			go oc.Flat.WatchSalt(context.Background())
		}
	}

	if len(output) == 0 {
		log.Fatal("No output settings")
	}

	iRBuf := dtap.NewRbuf(config.InputMsgBuffer, TotalRecvInputFrame, TotalLostInputFrame)
	fatalCh := make(chan error)

	outputCtx, outputCancel := context.WithCancel(context.Background())
	owg := &sync.WaitGroup{}
	for _, o := range output {
		child, _ := context.WithCancel(outputCtx)
		owg.Add(1)
		go func(o dtap.Output) {
			o.Run(child)
			owg.Done()
		}(o)
	}
	go outputLoop(output, iRBuf)

	inputCtx, intputCancel := context.WithCancel(context.Background())

	iwg := &sync.WaitGroup{}
	for _, i := range input {
		child, _ := context.WithCancel(inputCtx)
		iwg.Add(1)
		go func(i dtap.Input) {
			err := i.Run(child, iRBuf)
			if err != nil {
				log.Error(err)
				fatalCh <- err
			}
			iwg.Done()
		}(i)
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
