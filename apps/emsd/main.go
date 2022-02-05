package main

// Copyright (c) 2018 Bhojpur Consulting Private Limited, India. All rights reserved.

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/bhojpur/ems/pkg/core/lg"
	"github.com/bhojpur/ems/pkg/core/version"
	emssvr "github.com/bhojpur/ems/pkg/engine"
	"github.com/judwhite/go-svc"
	"github.com/mreiferson/go-options"
)

type program struct {
	once   sync.Once
	emssvr *emssvr.EMSD
}

func main() {
	prg := &program{}
	if err := svc.Run(prg, syscall.SIGINT, syscall.SIGTERM); err != nil {
		logFatal("%s", err)
	}
}

func (p *program) Init(env svc.Environment) error {
	opts := emssvr.NewOptions()

	flagSet := emsdFlagSet(opts)
	flagSet.Parse(os.Args[1:])

	rand.Seed(time.Now().UTC().UnixNano())

	if flagSet.Lookup("version").Value.(flag.Getter).Get().(bool) {
		fmt.Println(version.String("emsd"))
		os.Exit(0)
	}

	var cfg config
	configFile := flagSet.Lookup("config").Value.String()
	if configFile != "" {
		_, err := toml.DecodeFile(configFile, &cfg)
		if err != nil {
			logFatal("failed to load config file %s - %s", configFile, err)
		}
	}
	cfg.Validate()

	options.Resolve(opts, flagSet, cfg)

	emsd, err := emssvr.New(opts)
	if err != nil {
		logFatal("failed to instantiate emsd - %s", err)
	}
	p.emssvr = emsd

	return nil
}

func (p *program) Start() error {
	err := p.emssvr.LoadMetadata()
	if err != nil {
		logFatal("failed to load metadata - %s", err)
	}
	err = p.emssvr.PersistMetadata()
	if err != nil {
		logFatal("failed to persist metadata - %s", err)
	}

	go func() {
		err := p.emssvr.Main()
		if err != nil {
			p.Stop()
			os.Exit(1)
		}
	}()

	return nil
}

func (p *program) Stop() error {
	p.once.Do(func() {
		p.emssvr.Exit()
	})
	return nil
}

/*
func (p *program) Handle(s os.Signal) error {
	return svc.ErrStop
}
*/

// Context returns a context that will be canceled when emsd initiates the shutdown
func (p *program) Context() context.Context {
	return p.emssvr.Context()
}

func logFatal(f string, args ...interface{}) {
	lg.LogFatal("[emsd] ", f, args...)
}
