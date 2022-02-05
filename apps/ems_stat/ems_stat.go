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

// This is a utility application that polls /stats for all the producers
// of the specified topic/channel and displays aggregate stats

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bhojpur/ems/pkg/core/app"
	"github.com/bhojpur/ems/pkg/core/clusterinfo"
	"github.com/bhojpur/ems/pkg/core/http_api"
	"github.com/bhojpur/ems/pkg/core/version"
)

var (
	showVersion        = flag.Bool("version", false, "print version")
	topic              = flag.String("topic", "", "Bhojpur EMS topic")
	channel            = flag.String("channel", "", "Bhojpur EMS channel")
	interval           = flag.Duration("interval", 2*time.Second, "duration of time between polling/printing output")
	httpConnectTimeout = flag.Duration("http-client-connect-timeout", 2*time.Second, "timeout for HTTP connect")
	httpRequestTimeout = flag.Duration("http-client-request-timeout", 5*time.Second, "timeout for HTTP request")
	countNum           = numValue{}
	emsdHTTPAddrs      = app.StringArray{}
	lookupdHTTPAddrs   = app.StringArray{}
)

type numValue struct {
	isSet bool
	value int
}

func (nv *numValue) String() string { return "N" }

func (nv *numValue) Set(s string) error {
	value, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return err
	}
	nv.value = int(value)
	nv.isSet = true
	return nil
}

func init() {
	flag.Var(&emsdHTTPAddrs, "emsd-http-address", "emsd HTTP address (may be given multiple times)")
	flag.Var(&lookupdHTTPAddrs, "lookupd-http-address", "lookupd HTTP address (may be given multiple times)")
	flag.Var(&countNum, "count", "number of reports")
}

func statLoop(interval time.Duration, connectTimeout time.Duration, requestTimeout time.Duration,
	topic string, channel string, emsdTCPAddrs []string, lookupdHTTPAddrs []string) {
	ci := clusterinfo.New(nil, http_api.NewClient(nil, connectTimeout, requestTimeout))
	var o *clusterinfo.ChannelStats
	for i := 0; !countNum.isSet || countNum.value >= i; i++ {
		var producers clusterinfo.Producers
		var err error

		if len(lookupdHTTPAddrs) != 0 {
			producers, err = ci.GetLookupdTopicProducers(topic, lookupdHTTPAddrs)
		} else {
			producers, err = ci.GetEMSDTopicProducers(topic, emsdHTTPAddrs)
		}
		if err != nil {
			log.Fatalf("ERROR: failed to get topic producers - %s", err)
		}

		_, channelStats, err := ci.GetEMSDStats(producers, topic, channel, false)
		if err != nil {
			log.Fatalf("ERROR: failed to get emsd stats - %s", err)
		}

		c, ok := channelStats[channel]
		if !ok {
			log.Fatalf("ERROR: failed to find channel(%s) in stats metadata for topic(%s)", channel, topic)
		}

		if i%25 == 0 {
			fmt.Printf("%s+%s+%s\n",
				"------rate------",
				"----------------depth----------------",
				"--------------metadata---------------")
			fmt.Printf("%7s %7s | %7s %7s %7s %5s %5s | %7s %7s %12s %7s\n",
				"ingress", "egress",
				"total", "mem", "disk", "inflt",
				"def", "req", "t-o", "msgs", "clients")
		}

		if o == nil {
			o = c
			time.Sleep(interval)
			continue
		}

		// TODO: paused
		fmt.Printf("%7d %7d | %7d %7d %7d %5d %5d | %7d %7d %12d %7d\n",
			int64(float64(c.MessageCount-o.MessageCount)/interval.Seconds()),
			int64(float64(c.MessageCount-o.MessageCount-(c.Depth-o.Depth))/interval.Seconds()),
			c.Depth,
			c.MemoryDepth,
			c.BackendDepth,
			c.InFlightCount,
			c.DeferredCount,
			c.RequeueCount,
			c.TimeoutCount,
			c.MessageCount,
			c.ClientCount)

		o = c
		time.Sleep(interval)
	}
	os.Exit(0)
}

func checkAddrs(addrs []string) error {
	for _, a := range addrs {
		if strings.HasPrefix(a, "http") {
			return errors.New("address should not contain scheme")
		}
	}
	return nil
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("ems_stat v%s\n", version.Binary)
		return
	}

	if *topic == "" || *channel == "" {
		log.Fatal("--topic and --channel are required")
	}

	intvl := *interval
	if int64(intvl) <= 0 {
		log.Fatal("--interval should be positive")
	}

	connectTimeout := *httpConnectTimeout
	if int64(connectTimeout) <= 0 {
		log.Fatal("--http-client-connect-timeout should be positive")
	}

	requestTimeout := *httpRequestTimeout
	if int64(requestTimeout) <= 0 {
		log.Fatal("--http-client-request-timeout should be positive")
	}

	if countNum.isSet && countNum.value <= 0 {
		log.Fatal("--count should be positive")
	}

	if len(emsdHTTPAddrs) == 0 && len(lookupdHTTPAddrs) == 0 {
		log.Fatal("--emsd-http-address or --lookupd-http-address required")
	}
	if len(emsdHTTPAddrs) > 0 && len(lookupdHTTPAddrs) > 0 {
		log.Fatal("use --emsd-http-address or --lookupd-http-address not both")
	}

	if err := checkAddrs(emsdHTTPAddrs); err != nil {
		log.Fatalf("--emsd-http-address error - %s", err)
	}

	if err := checkAddrs(lookupdHTTPAddrs); err != nil {
		log.Fatalf("--lookupd-http-address error - %s", err)
	}

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	go statLoop(intvl, connectTimeout, requestTimeout, *topic, *channel, emsdHTTPAddrs, lookupdHTTPAddrs)

	<-termChan
}
