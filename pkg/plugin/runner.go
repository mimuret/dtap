package plugin

import (
	"context"
	"errors"
	"sync"

	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/mimuret/dtap/v3/pkg/types"
)

type Runner interface {
	types.Plugin
	Start(ctx context.Context, errCh chan<- error)
}

type InputRunner struct {
	types.InputPlugin
	// forwarder is used to write dnstap messages to the buffer
	*Forwarder
}

func NewInputRunner(cfg *config.InputBlock) (*InputRunner, error) {
	p, err := registry.CreateInputPlugin(cfg)
	if err != nil {
		return nil, err
	}
	return &InputRunner{
		InputPlugin: p,
		Forwarder:   NewForwarder(cfg.ConstLabels),
	}, nil
}

func SetupInputRunners(config []*config.InputBlock) (map[string]*InputRunner, error) {
	var errs error
	inputRunners := make(map[string]*InputRunner)
	for _, block := range config {
		runner, err := NewInputRunner(block)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		inputRunners[runner.GetFullName()] = runner
	}
	if errs != nil {
		return nil, errs
	}
	return inputRunners, nil
}

func (r *InputRunner) Start(ctx context.Context, errCh chan<- error) {
	if r.Forwarder == nil {
		errCh <- PluginError(r, "Forwarder is not set")
		return
	}
	err := r.InputPlugin.Start(ctx, r.Forwarder)
	if err != nil {
		errCh <- err
		return
	}
}

type FilterRunner struct {
	types.FilterPlugin
	// buffer for storing dnstap messages
	types.Buffer
	// forwarder is used to write dnstap messages to the buffer
	*Forwarder
}

func NewFilterRunner(cfg *config.FilterBlock) (*FilterRunner, error) {
	p, err := registry.CreateFilterPlugin(cfg)
	if err != nil {
		return nil, err
	}
	return &FilterRunner{
		FilterPlugin: p,
		Buffer:       NewBufferFromFilterPlugin(cfg),
		Forwarder:    NewForwarder(cfg.ConstLabels),
	}, nil
}

func SetupFilterRunners(config []*config.FilterBlock) (map[string]*FilterRunner, error) {
	var errs error
	filterRunners := make(map[string]*FilterRunner)
	for _, block := range config {
		runner, err := NewFilterRunner(block)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		filterRunners[runner.GetFullName()] = runner
	}
	if errs != nil {
		return nil, errs
	}
	return filterRunners, nil
}

func (r *FilterRunner) Start(ctx context.Context, errCh chan<- error) {
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				errCh <- ctx.Err()
			}
		case t := <-r.Buffer.Read():
			t = r.Filter(ctx, t)
			if t == nil {
				continue
			}
			r.Forwarder.Forward(t)
		}
	}
}

type OutputRunner struct {
	*config.OutputBlock
	outputPlugins []types.OutputPlugin
	// buffer for storing dnstap messages
	buf         types.Buffer
	constLabels map[string]string
}

func NewOutputRunner(cfg *config.OutputBlock) (*OutputRunner, error) {
	p, err := registry.CreateOutputPlugin(cfg)
	if err != nil {
		return nil, err
	}
	if p.MaxConcurrent() < cfg.GetConcurrency() {
		return nil, PluginError(p, "MaxConcurrent is less than configured concurrency: %d < %d", p.MaxConcurrent(), cfg.GetConcurrency())
	}
	outputPlugins := []types.OutputPlugin{p}
	for i := uint(1); i < cfg.GetConcurrency(); i++ {
		op, err := registry.CreateOutputPlugin(cfg)
		if err != nil {
			return nil, PluginError(p, "failed to create output plugin connection: %w", err)
		}
		outputPlugins = append(outputPlugins, op)
	}
	return &OutputRunner{
		OutputBlock:   cfg,
		outputPlugins: outputPlugins,
		buf:           NewBufferFromOutput(cfg),
		constLabels:   cfg.ConstLabels,
	}, nil
}

func SetupOutputRunners(config []*config.OutputBlock) (map[string]*OutputRunner, error) {
	var errs error
	outputRunners := make(map[string]*OutputRunner)
	for _, block := range config {
		runner, err := NewOutputRunner(block)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		outputRunners[runner.GetFullName()] = runner
	}
	if errs != nil {
		return nil, errs
	}
	return outputRunners, nil
}

func (r *OutputRunner) Start(ctx context.Context, errCh chan<- error) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	for _, plugin := range r.outputPlugins {
		wg.Add(1)
		go func(p types.OutputPlugin) {
			defer wg.Done()
			if err := p.Start(ctx, r.buf); err != nil {
				errCh <- err
			}
		}(plugin)
	}
}

func (r *OutputRunner) Write(m *types.DnstapMessage) {
	if r.constLabels != nil {
		for k, v := range r.constLabels {
			m.Labels[k] = v
		}
	}
	r.buf.Write(m)
}

func (r *OutputRunner) Read() <-chan *types.DnstapMessage {
	return r.buf.Read()
}
