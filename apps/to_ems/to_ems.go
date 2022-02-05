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

// This is a Bhojpur EMS client that publishes incoming messages from
// stdin to the specified topic.

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	emsctl "github.com/bhojpur/ems/pkg/client"
	"github.com/bhojpur/ems/pkg/core/app"
	"github.com/bhojpur/ems/pkg/core/version"
)

var (
	topic     = flag.String("topic", "", "Bhojpur EMS topic to publish to")
	delimiter = flag.String("delimiter", "\n", "character to split input from stdin")

	destEmsdTCPAddrs = app.StringArray{}
)

func init() {
	flag.Var(&destEmsdTCPAddrs, "emsd-tcp-address", "destination EMSd TCP address (may be given multiple times)")
}

func main() {
	cfg := emsctl.NewConfig()
	flag.Var(&emsctl.ConfigFlag{cfg}, "producer-opt", "option to passthrough to ems.Producer (may be given multiple times)")
	rate := flag.Int64("rate", 0, "Throttle messages to n/second. 0 to disable")

	flag.Parse()

	if len(*topic) == 0 {
		log.Fatal("--topic required")
	}

	if len(*delimiter) != 1 {
		log.Fatal("--delimiter must be a single byte")
	}

	stopChan := make(chan bool)
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	cfg.UserAgent = fmt.Sprintf("to_ems/%s emsctl/%s", version.Binary, emsctl.VERSION)

	// make the producers
	producers := make(map[string]*emsctl.Producer)
	for _, addr := range destEmsdTCPAddrs {
		producer, err := emsctl.NewProducer(addr, cfg)
		if err != nil {
			log.Fatalf("failed to create ems.Producer - %s", err)
		}
		producers[addr] = producer
	}

	if len(producers) == 0 {
		log.Fatal("--emsd-tcp-address required")
	}

	throttleEnabled := *rate >= 1
	balance := int64(1)
	// avoid divide by 0 if !throttleEnabled
	var interval time.Duration
	if throttleEnabled {
		interval = time.Second / time.Duration(*rate)
	}
	go func() {
		if !throttleEnabled {
			return
		}
		log.Printf("Throttling messages rate to max:%d/second", *rate)
		// every tick increase the number of messages we can send
		for _ = range time.Tick(interval) {
			n := atomic.AddInt64(&balance, 1)
			// if we build up more than 1s of capacity just bound to that
			if n > int64(*rate) {
				atomic.StoreInt64(&balance, int64(*rate))
			}
		}
	}()

	r := bufio.NewReader(os.Stdin)
	delim := (*delimiter)[0]
	go func() {
		for {
			var err error
			if throttleEnabled {
				currentBalance := atomic.LoadInt64(&balance)
				if currentBalance <= 0 {
					time.Sleep(interval)
				}
				err = readAndPublish(r, delim, producers)
				atomic.AddInt64(&balance, -1)
			} else {
				err = readAndPublish(r, delim, producers)
			}
			if err != nil {
				if err != io.EOF {
					log.Fatal(err)
				}
				close(stopChan)
				break
			}
		}
	}()

	select {
	case <-termChan:
	case <-stopChan:
	}

	for _, producer := range producers {
		producer.Stop()
	}
}

// readAndPublish reads to the delim from r and publishes the bytes
// to the map of producers.
func readAndPublish(r *bufio.Reader, delim byte, producers map[string]*emsctl.Producer) error {
	line, readErr := r.ReadBytes(delim)

	if len(line) > 0 {
		// trim the delimiter
		line = line[:len(line)-1]
	}

	if len(line) == 0 {
		return readErr
	}

	for _, producer := range producers {
		err := producer.Publish(*topic, line)
		if err != nil {
			return err
		}
	}

	return readErr
}
