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
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	emsctl "github.com/bhojpur/ems/pkg/client"
	"github.com/bhojpur/ems/pkg/core/app"
	"github.com/bhojpur/ems/pkg/core/version"
)

var (
	showVersion = flag.Bool("version", false, "print version string")

	channel       = flag.String("channel", "", "Bhojpur EMS channel")
	maxInFlight   = flag.Int("max-in-flight", 200, "max number of messages to allow in flight")
	totalMessages = flag.Int("n", 0, "total messages to show (will wait if starved)")
	printTopic    = flag.Bool("print-topic", false, "print topic name where message was received")

	emsdTCPAddrs     = app.StringArray{}
	lookupdHTTPAddrs = app.StringArray{}
	topics           = app.StringArray{}
)

func init() {
	flag.Var(&emsdTCPAddrs, "emsd-tcp-address", "EMSd TCP address (may be given multiple times)")
	flag.Var(&lookupdHTTPAddrs, "lookupd-http-address", "lookupd HTTP address (may be given multiple times)")
	flag.Var(&topics, "topic", "Bhojpur EMS topic (may be given multiple times)")
}

type TailHandler struct {
	topicName     string
	totalMessages int
	messagesShown int
}

func (th *TailHandler) HandleMessage(m *emsctl.Message) error {
	th.messagesShown++

	if *printTopic {
		_, err := os.Stdout.WriteString(th.topicName)
		if err != nil {
			log.Fatalf("ERROR: failed to write to os.Stdout - %s", err)
		}
		_, err = os.Stdout.WriteString(" | ")
		if err != nil {
			log.Fatalf("ERROR: failed to write to os.Stdout - %s", err)
		}
	}

	_, err := os.Stdout.Write(m.Body)
	if err != nil {
		log.Fatalf("ERROR: failed to write to os.Stdout - %s", err)
	}
	_, err = os.Stdout.WriteString("\n")
	if err != nil {
		log.Fatalf("ERROR: failed to write to os.Stdout - %s", err)
	}
	if th.totalMessages > 0 && th.messagesShown >= th.totalMessages {
		os.Exit(0)
	}
	return nil
}

func main() {
	cfg := emsctl.NewConfig()

	flag.Var(&emsctl.ConfigFlag{cfg}, "consumer-opt", "option to passthrough to ems.Consumer (may be given multiple times)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("ems_tail v%s\n", version.Binary)
		return
	}

	if *channel == "" {
		rand.Seed(time.Now().UnixNano())
		*channel = fmt.Sprintf("tail%06d#ephemeral", rand.Int()%999999)
	}

	if len(emsdTCPAddrs) == 0 && len(lookupdHTTPAddrs) == 0 {
		log.Fatal("--emsd-tcp-address or --lookupd-http-address required")
	}
	if len(emsdTCPAddrs) > 0 && len(lookupdHTTPAddrs) > 0 {
		log.Fatal("use --emsd-tcp-address or --lookupd-http-address not both")
	}
	if len(topics) == 0 {
		log.Fatal("--topic required")
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Don't ask for more messages than we want
	if *totalMessages > 0 && *totalMessages < *maxInFlight {
		*maxInFlight = *totalMessages
	}

	cfg.UserAgent = fmt.Sprintf("ems_tail/%s emsctl/%s", version.Binary, emsctl.VERSION)
	cfg.MaxInFlight = *maxInFlight

	consumers := []*emsctl.Consumer{}
	for i := 0; i < len(topics); i += 1 {
		log.Printf("Adding consumer for topic: %s\n", topics[i])

		consumer, err := emsctl.NewConsumer(topics[i], *channel, cfg)
		if err != nil {
			log.Fatal(err)
		}

		consumer.AddHandler(&TailHandler{topicName: topics[i], totalMessages: *totalMessages})

		err = consumer.ConnectToEMSDs(emsdTCPAddrs)
		if err != nil {
			log.Fatal(err)
		}

		err = consumer.ConnectToEMSLookupds(lookupdHTTPAddrs)
		if err != nil {
			log.Fatal(err)
		}

		consumers = append(consumers, consumer)
	}

	<-sigChan

	for _, consumer := range consumers {
		consumer.Stop()
	}
	for _, consumer := range consumers {
		<-consumer.StopChan
	}
}
