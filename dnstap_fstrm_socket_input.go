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
	"net"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var closeWant string = "use of closed network connection"

type DnstapFstrmSocketInput struct {
	listener net.Listener
	readDone chan struct{}
}

func NewDnstapFstrmSocketInput(listener net.Listener) (*DnstapFstrmSocketInput, error) {
	return &DnstapFstrmSocketInput{
		listener: listener,
		readDone: make(chan struct{}),
	}, nil
}

func (i *DnstapFstrmSocketInput) runRead(ctx context.Context, rbuf *RBuf, errCh chan error) {
	readCtx, readCancel := context.WithCancel(ctx)
	for {
		conn, err := i.listener.Accept()
		if err != nil {
			if !strings.Contains(err.Error(), closeWant) && log.GetLevel() >= log.InfoLevel {
				errCh <- errors.Wrapf(err, "can't accept unix socket")
			}
			readCancel()
			close(i.readDone)
			break
		}
		if log.GetLevel() >= log.DebugLevel {
			errCh <- errors.New("accept connection")
		}
		input, err := NewDnstapFstrmInput(conn, true)
		if err != nil {
			if log.GetLevel() >= log.InfoLevel {
				errCh <- errors.Wrapf(err, "can't create NewDnstapFstrmInput")
			}
			continue
		}
		childCtx, _ := context.WithCancel(readCtx)
		go input.Read(childCtx, rbuf, errCh)
	}
}

func (i *DnstapFstrmSocketInput) Run(ctx context.Context, rbuf *RBuf, errCh chan error) {
	childCtx, _ := context.WithCancel(ctx)
	go i.runRead(childCtx, rbuf, errCh)
	select {
	case <-ctx.Done():
		i.listener.Close()
	case <-i.ReadDone():
		break
	}
	log.Info("finish input")
}

func (i *DnstapFstrmSocketInput) ReadDone() <-chan struct{} {
	return i.readDone
}
