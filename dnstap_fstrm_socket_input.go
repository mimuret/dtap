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

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

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
	if log.GetLevel() >= log.DebugLevel {
		errCh <- errors.New("start socket input")
	}
	for {
		conn, err := i.listener.Accept()
		if err != nil {
			if log.GetLevel() >= log.InfoLevel {
				errCh <- errors.Wrapf(err, "can't accept unix socket")
			}
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
		go input.Read(ctx, rbuf, errCh)
	}
}

func (i *DnstapFstrmSocketInput) Run(ctx context.Context, rbuf *RBuf, errCh chan error) {
	go i.runRead(ctx, rbuf, errCh)
	select {
	case <-ctx.Done():
		i.listener.Close()
	case <-i.ReadDone():
		break
	}
}

func (i *DnstapFstrmSocketInput) ReadDone() <-chan struct{} {
	return i.readDone
}
