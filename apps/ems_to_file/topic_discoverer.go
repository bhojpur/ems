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
	"os"
	"regexp"
	"sync"
	"time"

	emsctl "github.com/bhojpur/ems/pkg/client"
	"github.com/bhojpur/ems/pkg/core/clusterinfo"
	"github.com/bhojpur/ems/pkg/core/http_api"
	"github.com/bhojpur/ems/pkg/core/lg"
)

type TopicDiscoverer struct {
	logf     lg.AppLogFunc
	opts     *Options
	ci       *clusterinfo.ClusterInfo
	topics   map[string]*FileLogger
	hupChan  chan os.Signal
	termChan chan os.Signal
	wg       sync.WaitGroup
	cfg      *emsctl.Config
}

func newTopicDiscoverer(logf lg.AppLogFunc, opts *Options, cfg *emsctl.Config, hupChan chan os.Signal, termChan chan os.Signal) *TopicDiscoverer {
	client := http_api.NewClient(nil, opts.HTTPClientConnectTimeout, opts.HTTPClientRequestTimeout)
	return &TopicDiscoverer{
		logf:     logf,
		opts:     opts,
		ci:       clusterinfo.New(nil, client),
		topics:   make(map[string]*FileLogger),
		hupChan:  hupChan,
		termChan: termChan,
		cfg:      cfg,
	}
}

func (t *TopicDiscoverer) updateTopics(topics []string) {
	for _, topic := range topics {
		if _, ok := t.topics[topic]; ok {
			continue
		}

		if !t.isTopicAllowed(topic) {
			t.logf(lg.WARN, "skipping topic %s (doesn't match pattern %s)", topic, t.opts.TopicPattern)
			continue
		}

		fl, err := NewFileLogger(t.logf, t.opts, topic, t.cfg)
		if err != nil {
			t.logf(lg.ERROR, "couldn't create logger for new topic %s: %s", topic, err)
			continue
		}
		t.topics[topic] = fl

		t.wg.Add(1)
		go func(fl *FileLogger) {
			fl.router()
			t.wg.Done()
		}(fl)
	}
}

func (t *TopicDiscoverer) run() {
	var ticker <-chan time.Time
	if len(t.opts.Topics) == 0 {
		ticker = time.Tick(t.opts.TopicRefreshInterval)
	}
	t.updateTopics(t.opts.Topics)
forloop:
	for {
		select {
		case <-ticker:
			newTopics, err := t.ci.GetLookupdTopics(t.opts.EMSLookupdHTTPAddrs)
			if err != nil {
				t.logf(lg.ERROR, "could not retrieve topic list: %s", err)
				continue
			}
			t.updateTopics(newTopics)
		case <-t.termChan:
			for _, fl := range t.topics {
				close(fl.termChan)
			}
			break forloop
		case <-t.hupChan:
			for _, fl := range t.topics {
				fl.hupChan <- true
			}
		}
	}
	t.wg.Wait()
}

func (t *TopicDiscoverer) isTopicAllowed(topic string) bool {
	if t.opts.TopicPattern == "" {
		return true
	}
	match, err := regexp.MatchString(t.opts.TopicPattern, topic)
	if err != nil {
		return false
	}
	return match
}
