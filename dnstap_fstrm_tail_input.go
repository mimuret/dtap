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

package dtap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/fsnotify/fsnotify"
)

type DnstapFstrmTailInput struct {
	config    *InputTailConfig
	readError chan error
	modifies  chan string
	readers   map[string]bool
}

func NewDnstapFstrmTailInput(config *InputTailConfig) *DnstapFstrmTailInput {
	i := &DnstapFstrmTailInput{
		config:    config,
		readError: make(chan error),
		modifies:  make(chan string, 128),
		readers:   map[string]bool{},
	}
	return i
}
func (i *DnstapFstrmTailInput) runSearchPath(ctx context.Context) error {
	var last time.Time = time.Now()
	mainTicker := time.NewTicker(time.Duration(60) * time.Second)
	for {
		matches, err := filepath.Glob(i.config.GetPath())
		if err != nil {
			return fmt.Errorf("search file error, path: %s err: %w", i.config.GetPath(), err)
		} else {
			for _, filename := range matches {
				if _, ok := i.readers[filename]; ok != false {
					if stat, err := os.Stat(filename); err != nil {
						if last.After(stat.ModTime()) {
							i.modifies <- filename
						}
					} else {
						log.Debugf("stat file error, %s, path: %s", err, filename)
					}
				}
			}
			last = time.Now()
		}
		select {
		case <-ctx.Done():
			return nil
		case <-mainTicker.C:
		}
	}
}
func (i *DnstapFstrmTailInput) runReadFile(ctx context.Context, filename string, rbuf *RBuf) error {
	modify := make(chan bool)
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file, path: %s err: %w", filename, err)
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	defer watcher.Close()
	watcher.Add(filename)
	subCtx, cancel := context.WithCancel(ctx)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					log.Debugf("watch failed, %s, path: %s", err, filename)
					cancel()
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					modify <- true
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					cancel()
					return
				}
				log.Debugf("watch failed, %s, path: %s", err, filename)
			}
		}
	}()
	input, err := NewDnstapFstrmInput(f, false)
	if err != nil {
		return err
	}
	for {
		input.Read(ctx, rbuf)
		timer := time.NewTimer(5 * time.Minute)
		select {
		case <-timer.C:
		case <-subCtx.Done():
		case <-ctx.Done():
			return nil
		case <-modify:
		}
	}
	delete(i.readers, filename)
	return nil
}
func (i *DnstapFstrmTailInput) Run(ctx context.Context, rbuf *RBuf) error {
	var err error
	childCtx, _ := context.WithCancel(ctx)
	go i.runSearchPath(childCtx)
L:
	for {
		select {
		case filename := <-i.modifies:
			i.readers[filename] = true
			childCtx, _ := context.WithCancel(ctx)
			go i.runReadFile(childCtx, filename, rbuf)
		case err = <-i.readError:
			break L
		case <-ctx.Done():
			break L
		}
	}
	return err
}
