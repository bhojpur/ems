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

// This is a client that writes out to a file, and optionally rolls the file

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	emsctl "github.com/bhojpur/ems/pkg/client"
	"github.com/bhojpur/ems/pkg/core/app"
	"github.com/bhojpur/ems/pkg/core/lg"
	"github.com/bhojpur/ems/pkg/core/version"
	"github.com/mreiferson/go-options"
)

func hasArg(s string) bool {
	argExist := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == s {
			argExist = true
		}
	})
	return argExist
}

func flagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("emsd", flag.ExitOnError)

	fs.Bool("version", false, "print version string")
	fs.String("log-level", "info", "set log verbosity: debug, info, warn, error, or fatal")
	fs.String("log-prefix", "[ems_to_file] ", "log message prefix")

	fs.String("channel", "ems_to_file", "Bhojpur EMS channel")
	fs.Int("max-in-flight", 200, "max number of messages to allow in flight")

	fs.String("output-dir", "/tmp", "directory to write output files to")
	fs.String("work-dir", "", "directory for in-progress files before moving to output-dir")
	fs.String("datetime-format", "%Y-%m-%d_%H", "strftime compatible format for <DATETIME> in filename format")
	fs.String("filename-format", "<TOPIC>.<HOST><REV>.<DATETIME>.log", "output filename format (<TOPIC>, <HOST>, <PID>, <DATETIME>, <REV> are replaced. <REV> is increased when file already exists)")
	fs.String("host-identifier", "", "value to output in log filename in place of hostname. <SHORT_HOST> and <HOSTNAME> are valid replacement tokens")
	fs.Int("gzip-level", 6, "gzip compression level (1-9, 1=BestSpeed, 9=BestCompression)")
	fs.Bool("gzip", false, "gzip output files.")
	fs.Bool("skip-empty-files", false, "skip writing empty files")
	fs.Duration("topic-refresh", time.Minute, "how frequently the topic list should be refreshed")
	fs.String("topic-pattern", "", "only log topics matching the following pattern")

	fs.Int64("rotate-size", 0, "rotate the file when it grows bigger than `rotate-size` bytes")
	fs.Duration("rotate-interval", 0, "rotate the file every duration")
	fs.Duration("sync-interval", 30*time.Second, "sync file to disk every duration")

	fs.Duration("http-client-connect-timeout", 2*time.Second, "timeout for HTTP connect")
	fs.Duration("http-client-request-timeout", 5*time.Second, "timeout for HTTP request")

	emsdTCPAddrs := app.StringArray{}
	lookupdHTTPAddrs := app.StringArray{}
	topics := app.StringArray{}
	consumerOpts := app.StringArray{}
	fs.Var(&emsdTCPAddrs, "emsd-tcp-address", "EMSd TCP address (may be given multiple times)")
	fs.Var(&lookupdHTTPAddrs, "lookupd-http-address", "lookupd HTTP address (may be given multiple times)")
	fs.Var(&topics, "topic", "Bhojpur EMS topic (may be given multiple times)")
	fs.Var(&consumerOpts, "consumer-opt", "option to passthrough to ems.Consumer (may be given multiple times)")

	return fs
}

func main() {
	fs := flagSet()
	fs.Parse(os.Args[1:])

	if args := fs.Args(); len(args) > 0 {
		log.Fatalf("unknown arguments: %s", args)
	}

	opts := NewOptions()
	options.Resolve(opts, fs, nil)

	logger := log.New(os.Stderr, opts.LogPrefix, log.Ldate|log.Ltime|log.Lmicroseconds)
	logLevel, err := lg.ParseLogLevel(opts.LogLevel)
	if err != nil {
		log.Fatal("--log-level is invalid")
	}
	logf := func(lvl lg.LogLevel, f string, args ...interface{}) {
		lg.Logf(logger, logLevel, lvl, f, args...)
	}

	if fs.Lookup("version").Value.(flag.Getter).Get().(bool) {
		fmt.Printf("ems_to_file v%s\n", version.Binary)
		return
	}

	if opts.Channel == "" {
		log.Fatal("--channel is required")
	}

	if opts.HTTPClientConnectTimeout <= 0 {
		log.Fatal("--http-client-connect-timeout should be positive")
	}

	if opts.HTTPClientRequestTimeout <= 0 {
		log.Fatal("--http-client-request-timeout should be positive")
	}

	if len(opts.EMSDTCPAddrs) == 0 && len(opts.EMSLookupdHTTPAddrs) == 0 {
		log.Fatal("--emsd-tcp-address or --lookupd-http-address required.")
	}
	if len(opts.EMSDTCPAddrs) != 0 && len(opts.EMSLookupdHTTPAddrs) != 0 {
		log.Fatal("use --emsd-tcp-address or --lookupd-http-address not both")
	}

	if opts.GZIPLevel < 1 || opts.GZIPLevel > 9 {
		log.Fatalf("invalid --gzip-level value (%d), should be 1-9", opts.GZIPLevel)
	}

	if len(opts.Topics) == 0 && len(opts.TopicPattern) == 0 {
		log.Fatal("--topic or --topic-pattern required")
	}

	if len(opts.Topics) == 0 && len(opts.EMSLookupdHTTPAddrs) == 0 {
		log.Fatal("--lookupd-http-address must be specified when no --topic specified")
	}

	if opts.WorkDir == "" {
		opts.WorkDir = opts.OutputDir
	}

	cfg := emsctl.NewConfig()
	cfgFlag := emsctl.ConfigFlag{cfg}
	for _, opt := range opts.ConsumerOpts {
		cfgFlag.Set(opt)
	}
	cfg.UserAgent = fmt.Sprintf("ems_to_file/%s emsctl/%s", version.Binary, emsctl.VERSION)
	cfg.MaxInFlight = opts.MaxInFlight

	hupChan := make(chan os.Signal, 1)
	termChan := make(chan os.Signal, 1)
	signal.Notify(hupChan, syscall.SIGHUP)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	discoverer := newTopicDiscoverer(logf, opts, cfg, hupChan, termChan)
	discoverer.run()
}
