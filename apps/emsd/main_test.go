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
	"crypto/tls"
	"os"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/bhojpur/ems/pkg/core/lg"
	"github.com/bhojpur/ems/pkg/core/test"
	emssvr "github.com/bhojpur/ems/pkg/engine"
	"github.com/mreiferson/go-options"
)

func TestConfigFlagParsing(t *testing.T) {
	opts := emssvr.NewOptions()
	opts.Logger = test.NewTestLogger(t)

	flagSet := emsdFlagSet(opts)
	flagSet.Parse([]string{})

	var cfg config
	f, err := os.Open("../../contrib/emsd.cfg.example")
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer f.Close()
	toml.DecodeReader(f, &cfg)
	cfg["log_level"] = "debug"
	cfg.Validate()

	options.Resolve(opts, flagSet, cfg)
	emssvr.New(opts)

	if opts.TLSMinVersion != tls.VersionTLS10 {
		t.Errorf("min %#v not expected %#v", opts.TLSMinVersion, tls.VersionTLS10)
	}
	if opts.LogLevel != lg.DEBUG {
		t.Fatalf("log level: want debug, got %s", opts.LogLevel.String())
	}
}
