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
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/fsnotify/fsnotify"
)

type DnstapFstrmTailInput struct {
	config   *InputTailConfig
	modifies chan string
	readers  map[string]bool
}

func NewDnstapFstrmTailInput(config *InputTailConfig) *DnstapFstrmTailInput {
	i := &DnstapFstrmTailInput{
		config:   config,
		modifies: make(chan string, 128),
		readers:  map[string]bool{},
	}
	return i
}
func (i *DnstapFstrmTailInput) runSearchPath(ctx context.Context, errCh chan error) {
	var last time.Time = time.Now()
	mainTicker := time.NewTicker(time.Duration(60) * time.Second)
	for {
		matches, err := filepath.Glob(i.config.GetPath())
		if err != nil {
			if log.GetLevel() >= log.InfoLevel {

				errCh <- errors.Wrapf(err, "search file error, path: %s", i.config.GetPath())
			}
		} else {
			for _, filename := range matches {
				if _, ok := i.readers[filename]; ok != false {
					if stat, err := os.Stat(filename); err != nil {
						if last.After(stat.ModTime()) {
							i.modifies <- filename
						}
					} else {
						if log.GetLevel() >= log.DebugLevel {

							errCh <- errors.Wrapf(err, "stat file error, path: %s", filename)
						}
					}
				}
			}
			last = time.Now()
		}
		select {
		case <-ctx.Done():
			return
		case <-mainTicker.C:
		}
	}
}
func (i *DnstapFstrmTailInput) runReadFile(ctx context.Context, filename string, rbuf *RBuf, errCh chan error) {
	modify := make(chan bool)
	f, err := os.Open(filename)
	if err != nil {
		if log.GetLevel() >= log.InfoLevel {

			errCh <- errors.Wrapf(err, "can't open file, path: %s", filename)
		}
		return
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		if log.GetLevel() >= log.InfoLevel {

			errCh <- errors.Wrapf(err, "can't create file watcher")
		}
		return
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
					if log.GetLevel() >= log.DebugLevel {

						errCh <- errors.Wrapf(err, "watch failed, path: %s", filename)
					}
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
				if log.GetLevel() >= log.DebugLevel {
					errCh <- errors.Wrapf(err, "watch error path: %s", filename)
				}
			}
		}
	}()
	input, err := NewDnstapFstrmInput(f, false)
	if err != nil {
		return
	}
	for {
		input.Read(ctx, rbuf, errCh)
		timer := time.NewTimer(5 * time.Minute)
		select {
		case <-timer.C:
		case <-subCtx.Done():
		case <-ctx.Done():
			return
		case <-modify:
		}
	}
	delete(i.readers, filename)
}
func (i *DnstapFstrmTailInput) Run(ctx context.Context, rbuf *RBuf, errCh chan error) {
	childCtx, _ := context.WithCancel(ctx)
	go i.runSearchPath(childCtx, errCh)
	for filename := range i.modifies {
		i.readers[filename] = true
		childCtx, _ := context.WithCancel(ctx)
		go i.runReadFile(childCtx, filename, rbuf, errCh)
	}
}
