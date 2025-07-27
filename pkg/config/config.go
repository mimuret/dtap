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

package config

import (
	"fmt"
	"path/filepath"
	"strings"

	_ "github.com/go-viper/encoding/hcl"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/spf13/afero"

	"errors"
)

func blockError(block ConfigBlock, format string, args ...interface{}) error {
	return fmt.Errorf("plugin %s: %s", block.GetFullName(), fmt.Sprintf(format, args...))
}

func NewInputBlock(ptype, name, hclString string) (*InputBlock, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(hclString), "input.hcl")
	if diags.HasErrors() {
		return nil, errors.Join(diags.Errs()...)
	}
	block := &InputBlock{
		Type: ptype,
		Name: name,
		Body: file.Body,
	}
	return block, nil
}

type ConfigBlock interface {
	GetFullName() string
}

type InputBlock struct {
	Type            string            `hcl:"type,label"`
	Name            string            `hcl:"name,label"`
	Body            hcl.Body          `hcl:",remain"`
	ForwardToString []string          `hcl:"forward_to,attr"`
	ConstLabels     map[string]string `hcl:"labels,optional"`
}

func (b *InputBlock) GetType() string {
	return b.Type
}

func (b *InputBlock) GetName() string {
	return b.Name
}

func (b *InputBlock) GetFullName() string {
	return fmt.Sprintf("input.%s.%s", b.Type, b.Name)
}

func NewFilterBlock(ptype, name, hclString string) (*FilterBlock, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(hclString), "filter.hcl")
	if diags.HasErrors() {
		return nil, errors.Join(diags.Errs()...)
	}
	block := &FilterBlock{
		Type: ptype,
		Name: name,
		Body: file.Body,
	}
	return block, nil
}

type FilterBlock struct {
	Type            string            `hcl:"type,label"`
	Name            string            `hcl:"name,label"`
	Body            hcl.Body          `hcl:",remain"`
	BufferSize      uint              `hcl:"buffer_size,optional"`
	ForwardToString []string          `hcl:"forward_to,attr"`
	ConstLabels     map[string]string `hcl:"labels,optional"`
}

func (b *FilterBlock) GetType() string {
	return b.Type
}

func (b *FilterBlock) GetName() string {
	return b.Name
}

func (b *FilterBlock) GetFullName() string {
	return fmt.Sprintf("filter.%s.%s", b.Type, b.Name)
}

func (b *FilterBlock) GetBufferSize() uint {
	return b.BufferSize
}

func NewOutputBlock(ptype, name, hclString string) (*OutputBlock, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(hclString), "output.hcl")
	if diags.HasErrors() {
		return nil, errors.Join(diags.Errs()...)
	}
	block := &OutputBlock{
		Type: ptype,
		Name: name,
		Body: file.Body,
	}
	return block, nil
}

type OutputBlock struct {
	Type        string            `hcl:"type,label"`
	Name        string            `hcl:"name,label"`
	Body        hcl.Body          `hcl:",remain"`
	BufferSize  uint              `hcl:"buffer_size,optional"`
	Concurrency uint              `hcl:"concurrency,optional"`
	ConstLabels map[string]string `hcl:"labels,optional"`
}

func (b *OutputBlock) GetType() string {
	return b.Type
}

func (b *OutputBlock) GetName() string {
	return b.Name
}

func (b *OutputBlock) GetFullName() string {
	return fmt.Sprintf("output.%s.%s", b.Type, b.Name)
}

func (b *OutputBlock) GetBufferSize() uint {
	return b.BufferSize
}

func (b *OutputBlock) GetConcurrency() uint {
	if b.Concurrency == 0 {
		return 1 // Default concurrency is 1 if not specified
	}
	return b.Concurrency
}

func checkDuplicate[T ConfigBlock](data []T) error {
	var res error
	seen := make(map[string]struct{})
	for _, block := range data {
		key := block.GetFullName()
		if _, exists := seen[key]; exists {
			res = errors.Join(res, fmt.Errorf("duplicate config block found: %s", key))
		}

		seen[key] = struct{}{}
	}
	return res
}

// Config represents the entire configuration for the application.
type Config struct {
	// Input configuration.
	InputBlocks []*InputBlock `hcl:"input,block"`

	// Filter configuration.
	FilterBlocks []*FilterBlock `hcl:"filter,block"`

	// Output configuration.
	OutputBlocks []*OutputBlock `hcl:"output,block"`
}

func (c *Config) checkDuplicate() error {
	var res error
	if err := checkDuplicate(c.InputBlocks); err != nil {
		res = errors.Join(res, fmt.Errorf("input blocks validation failed: %w", err))
	}
	if err := checkDuplicate(c.FilterBlocks); err != nil {
		res = errors.Join(res, fmt.Errorf("filter blocks validation failed: %w", err))
	}
	if err := checkDuplicate(c.OutputBlocks); err != nil {
		res = errors.Join(res, fmt.Errorf("output blocks validation failed: %w", err))
	}
	return res
}

func detectLoop(plugins map[string][]string, start string, visited map[string]bool, stack map[string]bool) error {
	if stack[start] {
		return fmt.Errorf("loop detected in forward_to chain: %s", start)
	}
	if visited[start] {
		return nil
	}
	visited[start] = true
	stack[start] = true
	for _, next := range plugins[start] {
		if err := detectLoop(plugins, next, visited, stack); err != nil {
			return err
		}
	}
	stack[start] = false
	return nil
}

func (c *Config) checkForwardTo() error {
	var res error
	plugins := map[string]struct{}{}
	pluginGraph := map[string][]string{}

	// Register output blocks
	for _, block := range c.OutputBlocks {
		plugins[block.GetFullName()] = struct{}{}
	}

	// Register filter blocks
	for _, block := range c.FilterBlocks {
		plugins[block.GetFullName()] = struct{}{}
	}

	// Register input blocks
	for _, block := range c.InputBlocks {
		plugins[block.GetFullName()] = struct{}{}
	}

	// Validate input blocks
	for _, block := range c.InputBlocks {
		if len(block.ForwardToString) == 0 {
			res = errors.Join(res, blockError(block, "forward_to must not be empty"))
		}
		for _, forwardTo := range block.ForwardToString {
			if strings.HasPrefix(forwardTo, "input") {
				res = errors.Join(res, blockError(block, "forward_to `%s` is not supported", forwardTo))
			}
			if _, ok := plugins[forwardTo]; !ok {
				res = errors.Join(res, blockError(block, "forward_to `%s` does not exist", forwardTo))
			}
		}
	}

	// Validate filter blocks
	for _, block := range c.FilterBlocks {
		if len(block.ForwardToString) == 0 {
			res = errors.Join(res, blockError(block, "forward_to must not be empty"))
		}
		for _, forwardTo := range block.ForwardToString {
			if strings.HasPrefix(forwardTo, "input") {
				res = errors.Join(res, blockError(block, "forward_to `%s` is not supported", forwardTo))
			}
			if _, ok := plugins[forwardTo]; !ok {
				res = errors.Join(res, blockError(block, "forward_to `%s` does not exist", forwardTo))
			}
			// Build the plugin graph for loop detection
			pluginGraph[block.GetFullName()] = append(pluginGraph[block.GetFullName()], forwardTo)
		}
	}

	// Detect loops in filter chains
	visited := map[string]bool{}
	stack := map[string]bool{}
	for plugin := range pluginGraph {
		if err := detectLoop(pluginGraph, plugin, visited, stack); err != nil {
			res = errors.Join(res, err)
		}
	}

	return res
}

func LoadConfig(fs afero.Fs, cfgDir string) (*Config, error) {
	stat, err := fs.Stat(cfgDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", cfgDir, err)
	}
	var files []string
	if stat.IsDir() {
		files, err = afero.Glob(fs, filepath.Join(cfgDir, "*.hcl"))
		if err != nil {
			return nil, fmt.Errorf("failed to read config dir %s: %w", cfgDir, err)
		}
	} else {
		files = []string{cfgDir}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no HCL files found in directory %s", cfgDir)
	}
	// HCLパーサーを作成
	parser := hclparse.NewParser()
	hcFiles := make([]*hcl.File, 0, len(files))
	for _, file := range files {
		bs, err := afero.ReadFile(fs, file)
		if err != nil {
			return nil, fmt.Errorf("failed to open config file: %w", err)
		}
		// HCLファイルをパース
		hclFile, diags := parser.ParseHCL(bs, file)
		if diags.HasErrors() {
			return nil, fmt.Errorf("failed to parse HCL: %w", errors.Join(diags.Errs()...))
		}
		hcFiles = append(hcFiles, hclFile)
	}

	// 構造体にデコード
	var c Config
	diags := gohcl.DecodeBody(hcl.MergeFiles(hcFiles), nil, &c)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode HCL into struct: %w", errors.Join(diags.Errs()...))
	}
	// 重複チェック
	if err := c.checkDuplicate(); err != nil {
		return nil, err
	}

	// forward_to check
	if err := c.checkForwardTo(); err != nil {
		return nil, err
	}
	return &c, nil
}
