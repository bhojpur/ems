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

package client

/*
It is the official Go package for the Bhojpur EMS. It provides high-level
Consumer and Producer types as well as low-level functions to communicate
over the Bhojpur EMS protocol.

Consumer

Consuming messages from Bhojpur EMS can be done by creating an instance of
a Consumer and supplying it a handler.

	package main
	import (
		"log"
		"os/signal"
		ems "github.com/bhojpur/ems/pkg/client"
	)

	type myMessageHandler struct {}

	// HandleMessage implements the Handler interface.
	func (h *myMessageHandler) HandleMessage(m *ems.Message) error {
		if len(m.Body) == 0 {
			// Returning nil will automatically send a FIN command to Bhojpur EMS to
			// mark the message as processed. In this case, a message with an empty
			// body is simply ignored/discarded.
			return nil
		}

		// do whatever actual message processing is desired
		err := processMessage(m.Body)

		// Returning a non-nil error will automatically send a REQ command to
		// Bhojpur EMS to re-queue the message.
		return err
	}

	func main() {
		// Instantiate a consumer that will subscribe to the provided channel.
		config := ems.NewConfig()
		consumer, err := ems.NewConsumer("topic", "channel", config)
		if err != nil {
			log.Fatal(err)
		}

		// Set the Handler for messages received by this Consumer. Can be called
		// multiple times. See also AddConcurrentHandlers.
		consumer.AddHandler(&myMessageHandler{})

		// Use emslookupd to discover emsd instances.
		// See also ConnectToEMSD, ConnectToEMSDs, ConnectToEMSLookupds.
		err = consumer.ConnectToEMSLookupd("localhost:4161")
		if err != nil {
			log.Fatal(err)
		}

		// wait for signal to exit
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		// Gracefully stop the consumer.
		consumer.Stop()
	}

Producer

Producing messages can be done by creating an instance of a Producer.

	// Instantiate a producer.
	config := ems.NewConfig()
	producer, err := ems.NewProducer("127.0.0.1:4150", config)
	if err != nil {
		log.Fatal(err)
	}

	messageBody := []byte("hello")
	topicName := "topic"

	// Synchronously publish a single message to the specified topic.
	// Messages can also be sent asynchronously and/or in batches.
	err = producer.Publish(topicName, messageBody)
	if err != nil {
		log.Fatal(err)
	}

	// Gracefully stop the producer when appropriate (e.g. before shutting down the service)
	producer.Stop()

*/
