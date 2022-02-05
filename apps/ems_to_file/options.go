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

import "time"

type Options struct {
	Topics               []string      `flag:"topic"`
	TopicPattern         string        `flag:"topic-pattern"`
	TopicRefreshInterval time.Duration `flag:"topic-refresh"`
	Channel              string        `flag:"channel"`

	EMSDTCPAddrs             []string      `flag:"emsd-tcp-address"`
	EMSLookupdHTTPAddrs      []string      `flag:"lookupd-http-address"`
	ConsumerOpts             []string      `flag:"consumer-opt"`
	MaxInFlight              int           `flag:"max-in-flight"`
	HTTPClientConnectTimeout time.Duration `flag:"http-client-connect-timeout"`
	HTTPClientRequestTimeout time.Duration `flag:"http-client-request-timeout"`

	LogPrefix      string        `flag:"log-prefix"`
	LogLevel       string        `flag:"log-level"`
	OutputDir      string        `flag:"output-dir"`
	WorkDir        string        `flag:"work-dir"`
	DatetimeFormat string        `flag:"datetime-format"`
	FilenameFormat string        `flag:"filename-format"`
	HostIdentifier string        `flag:"host-identifier"`
	GZIPLevel      int           `flag:"gzip-level"`
	GZIP           bool          `flag:"gzip"`
	SkipEmptyFiles bool          `flag:"skip-empty-files"`
	RotateSize     int64         `flag:"rotate-size"`
	RotateInterval time.Duration `flag:"rotate-interval"`
	SyncInterval   time.Duration `flag:"sync-interval"`
}

func NewOptions() *Options {
	return &Options{
		LogPrefix:                "[ems_to_file] ",
		LogLevel:                 "info",
		Channel:                  "ems_to_file",
		MaxInFlight:              200,
		OutputDir:                "/tmp",
		DatetimeFormat:           "%Y-%m-%d_%H",
		FilenameFormat:           "<TOPIC>.<HOST><REV>.<DATETIME>.log",
		GZIPLevel:                6,
		TopicRefreshInterval:     time.Minute,
		SyncInterval:             30 * time.Second,
		HTTPClientConnectTimeout: 2 * time.Second,
		HTTPClientRequestTimeout: 5 * time.Second,
	}
}
