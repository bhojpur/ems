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
	"net"
	"sync"
	"time"

	emsctl "github.com/bhojpur/ems/pkg/client"
)

var (
	num        = flag.Int("num", 10000, "num channels")
	tcpAddress = flag.String("emsd-tcp-address", "127.0.0.1:4150", "<addr>:<port> to connect to Bhojpur EMS daemon")
)

func main() {
	flag.Parse()
	var wg sync.WaitGroup

	goChan := make(chan int)
	rdyChan := make(chan int)
	for j := 0; j < *num; j++ {
		wg.Add(1)
		go func(id int) {
			subWorker(*num, *tcpAddress, fmt.Sprintf("t%d", j), "ch", rdyChan, goChan, id)
			wg.Done()
		}(j)
		<-rdyChan
		time.Sleep(5 * time.Millisecond)
	}

	close(goChan)
	wg.Wait()
}

func subWorker(n int, tcpAddr string,
	topic string, channel string,
	rdyChan chan int, goChan chan int, id int) {
	conn, err := net.DialTimeout("tcp", tcpAddr, time.Second)
	if err != nil {
		panic(err.Error())
	}
	conn.Write(emsctl.MagicV2)
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	ci := make(map[string]interface{})
	ci["client_id"] = "test"
	cmd, _ := emsctl.Identify(ci)
	cmd.WriteTo(rw)
	emsctl.Subscribe(topic, channel).WriteTo(rw)
	rdyCount := 1
	rdy := rdyCount
	rdyChan <- 1
	<-goChan
	emsctl.Ready(rdyCount).WriteTo(rw)
	rw.Flush()
	emsctl.ReadResponse(rw)
	emsctl.ReadResponse(rw)
	for {
		resp, err := emsctl.ReadResponse(rw)
		if err != nil {
			panic(err.Error())
		}
		frameType, data, err := emsctl.UnpackResponse(resp)
		if err != nil {
			panic(err.Error())
		}
		if frameType == emsctl.FrameTypeError {
			panic(string(data))
		} else if frameType == emsctl.FrameTypeResponse {
			emsctl.Nop().WriteTo(rw)
			rw.Flush()
			continue
		}
		msg, err := emsctl.DecodeMessage(data)
		if err != nil {
			panic(err.Error())
		}
		emsctl.Finish(msg.ID).WriteTo(rw)
		rdy--
		if rdy == 0 {
			emsctl.Ready(rdyCount).WriteTo(rw)
			rdy = rdyCount
			rw.Flush()
		}
	}
}
