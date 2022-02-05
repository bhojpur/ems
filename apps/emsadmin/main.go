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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/BurntSushi/toml"
	emsadm "github.com/bhojpur/ems/pkg/admin"
	"github.com/bhojpur/ems/pkg/core/app"
	"github.com/bhojpur/ems/pkg/core/lg"
	"github.com/bhojpur/ems/pkg/core/version"
	"github.com/judwhite/go-svc"
	"github.com/mreiferson/go-options"
)

func emsadminFlagSet(opts *emsadm.Options) *flag.FlagSet {
	flagSet := flag.NewFlagSet("emsadmin", flag.ExitOnError)

	flagSet.String("config", "", "path to config file")
	flagSet.Bool("version", false, "print version string")

	logLevel := opts.LogLevel
	flagSet.Var(&logLevel, "log-level", "set log verbosity: debug, info, warn, error, or fatal")
	flagSet.String("log-prefix", "[emsadmin] ", "log message prefix")
	flagSet.Bool("verbose", false, "[deprecated] has no effect, use --log-level")

	flagSet.String("http-address", opts.HTTPAddress, "<addr>:<port> to listen on for HTTP clients")
	flagSet.String("base-path", opts.BasePath, "URL base path")
	flagSet.String("dev-static-dir", opts.DevStaticDir, "(development use only)")

	flagSet.String("graphite-url", opts.GraphiteURL, "graphite HTTP address")
	flagSet.Bool("proxy-graphite", false, "proxy HTTP requests to graphite")

	flagSet.String("statsd-counter-format", opts.StatsdCounterFormat, "The counter stats key formatting applied by the implementation of statsd. If no formatting is desired, set this to an empty string.")
	flagSet.String("statsd-gauge-format", opts.StatsdGaugeFormat, "The gauge stats key formatting applied by the implementation of statsd. If no formatting is desired, set this to an empty string.")
	flagSet.String("statsd-prefix", opts.StatsdPrefix, "prefix used for keys sent to statsd (%s for host replacement, must match emsd)")
	flagSet.Duration("statsd-interval", opts.StatsdInterval, "time interval emsd is configured to push to statsd (must match emsd)")

	flagSet.String("notification-http-endpoint", "", "HTTP endpoint (fully qualified) to which POST notifications of admin actions will be sent")

	flagSet.Duration("http-client-connect-timeout", opts.HTTPClientConnectTimeout, "timeout for HTTP connect")
	flagSet.Duration("http-client-request-timeout", opts.HTTPClientRequestTimeout, "timeout for HTTP request")

	flagSet.Bool("http-client-tls-insecure-skip-verify", false, "configure the HTTP client to skip verification of TLS certificates")
	flagSet.String("http-client-tls-root-ca-file", "", "path to CA file for the HTTP client")
	flagSet.String("http-client-tls-cert", "", "path to certificate file for the HTTP client")
	flagSet.String("http-client-tls-key", "", "path to key file for the HTTP client")

	flagSet.String("allow-config-from-cidr", opts.AllowConfigFromCIDR, "A CIDR from which to allow HTTP requests to the /config endpoint")
	flagSet.String("acl-http-header", opts.AclHttpHeader, "HTTP header to check for authenticated admin users")

	emslookupdHTTPAddresses := app.StringArray{}
	flagSet.Var(&emslookupdHTTPAddresses, "lookupd-http-address", "lookupd HTTP address (may be given multiple times)")
	emsdHTTPAddresses := app.StringArray{}
	flagSet.Var(&emsdHTTPAddresses, "emsd-http-address", "EMSd HTTP address (may be given multiple times)")
	adminUsers := app.StringArray{}
	flagSet.Var(&adminUsers, "admin-user", "admin user (may be given multiple times; if specified, only these users will be able to perform privileged actions; acl-http-header is used to determine the authenticated user)")

	return flagSet
}

type program struct {
	once     sync.Once
	emsadmin *emsadm.EMSAdmin
}

func main() {
	prg := &program{}
	if err := svc.Run(prg, syscall.SIGINT, syscall.SIGTERM); err != nil {
		logFatal("%s", err)
	}
}

func (p *program) Init(env svc.Environment) error {
	if env.IsWindowsService() {
		dir := filepath.Dir(os.Args[0])
		return os.Chdir(dir)
	}
	return nil
}

func (p *program) Start() error {
	opts := emsadm.NewOptions()

	flagSet := emsadminFlagSet(opts)
	flagSet.Parse(os.Args[1:])

	if flagSet.Lookup("version").Value.(flag.Getter).Get().(bool) {
		fmt.Println(version.String("emsadmin"))
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
	emsadmin, err := emsadm.New(opts)
	if err != nil {
		logFatal("failed to instantiate emsadmin - %s", err)
	}
	p.emsadmin = emsadmin

	go func() {
		err := p.emsadmin.Main()
		if err != nil {
			p.Stop()
			os.Exit(1)
		}
	}()

	return nil
}

func (p *program) Stop() error {
	p.once.Do(func() {
		p.emsadmin.Exit()
	})
	return nil
}

func logFatal(f string, args ...interface{}) {
	lg.LogFatal("[emsadmin] ", f, args...)
}
