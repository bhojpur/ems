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
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	emsctl "github.com/bhojpur/ems/pkg/client"
)

var (
	runfor     = flag.Duration("runfor", 10*time.Second, "duration of time to run")
	tcpAddress = flag.String("emsd-tcp-address", "127.0.0.1:4150", "<addr>:<port> to connect to Bhojpur EMS daemon")
	size       = flag.Int("size", 200, "size of messages")
	topic      = flag.String("topic", "sub_bench", "topic to receive messages on")
	channel    = flag.String("channel", "ch", "channel to receive messages on")
	deadline   = flag.String("deadline", "", "deadline to start the benchmark run")
	rdy        = flag.Int("rdy", 2500, "RDY count to use")
)

var totalMsgCount int64

func main() {
	flag.Parse()
	var wg sync.WaitGroup

	log.SetPrefix("[bench_reader] ")

	goChan := make(chan int)
	rdyChan := make(chan int)
	workers := runtime.GOMAXPROCS(0)
	for j := 0; j < workers; j++ {
		wg.Add(1)
		go func(id int) {
			subWorker(*runfor, workers, *tcpAddress, *topic, *channel, rdyChan, goChan, id)
			wg.Done()
		}(j)
		<-rdyChan
	}

	if *deadline != "" {
		t, err := time.Parse("2006-01-02 15:04:05", *deadline)
		if err != nil {
			log.Fatal(err)
		}
		d := t.Sub(time.Now())
		log.Printf("sleeping until %s (%s)", t, d)
		time.Sleep(d)
	}

	start := time.Now()
	close(goChan)
	wg.Wait()
	end := time.Now()
	duration := end.Sub(start)
	tmc := atomic.LoadInt64(&totalMsgCount)
	log.Printf("duration: %s - %.03fmb/s - %.03fops/s - %.03fus/op",
		duration,
		float64(tmc*int64(*size))/duration.Seconds()/1024/1024,
		float64(tmc)/duration.Seconds(),
		float64(duration/time.Microsecond)/float64(tmc))
}

func subWorker(td time.Duration, workers int, tcpAddr string, topic string, channel string, rdyChan chan int, goChan chan int, id int) {
	conn, err := net.DialTimeout("tcp", tcpAddr, time.Second)
	if err != nil {
		panic(err.Error())
	}
	conn.Write(emsctl.MagicV2)
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	ci := make(map[string]interface{})
	ci["client_id"] = "reader"
	ci["hostname"] = "reader"
	ci["user_agent"] = fmt.Sprintf("bench_reader/%s", emsctl.VERSION)
	cmd, _ := emsctl.Identify(ci)
	cmd.WriteTo(rw)
	emsctl.Subscribe(topic, channel).WriteTo(rw)
	rdyChan <- 1
	<-goChan
	emsctl.Ready(*rdy).WriteTo(rw)
	rw.Flush()
	emsctl.ReadResponse(rw)
	emsctl.ReadResponse(rw)
	var msgCount int64
	go func() {
		time.Sleep(td)
		conn.Close()
	}()
	for {
		resp, err := emsctl.ReadResponse(rw)
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				break
			}
			panic(err.Error())
		}
		frameType, data, err := emsctl.UnpackResponse(resp)
		if err != nil {
			panic(err.Error())
		}
		if frameType == emsctl.FrameTypeError {
			panic(string(data))
		} else if frameType == emsctl.FrameTypeResponse {
			continue
		}
		msg, err := emsctl.DecodeMessage(data)
		if err != nil {
			panic(err.Error())
		}
		emsctl.Finish(msg.ID).WriteTo(rw)
		msgCount++
		if float64(msgCount%int64(*rdy)) > float64(*rdy)*0.75 {
			rw.Flush()
		}
	}
	atomic.AddInt64(&totalMsgCount, msgCount)
}
