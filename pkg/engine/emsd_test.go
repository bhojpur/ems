package engine

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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bhojpur/ems/pkg/core/http_api"
	"github.com/bhojpur/ems/pkg/core/test"
	emslookupd "github.com/bhojpur/ems/pkg/lookup"
)

const (
	ConnectTimeout = 2 * time.Second
	RequestTimeout = 5 * time.Second
)

func getMetadata(n *EMSD) (*meta, error) {
	fn := newMetadataFile(n.getOpts())
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}

	var m meta
	err = json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func TestStartup(t *testing.T) {
	var msg *Message

	iterations := 300
	doneExitChan := make(chan int)

	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	opts.MemQueueSize = 100
	opts.MaxBytesPerFile = 10240
	_, _, emsd := mustStartEMSD(opts)
	defer os.RemoveAll(opts.DataPath)

	origDataPath := opts.DataPath

	topicName := "emsd_test" + strconv.Itoa(int(time.Now().Unix()))

	exitChan := make(chan int)
	go func() {
		<-exitChan
		emsd.Exit()
		doneExitChan <- 1
	}()

	// verify emsd metadata shows no topics
	err := emsd.PersistMetadata()
	test.Nil(t, err)
	atomic.StoreInt32(&emsd.isLoading, 1)
	emsd.GetTopic(topicName) // will not persist if `flagLoading`
	m, err := getMetadata(emsd)
	test.Nil(t, err)
	test.Equal(t, 0, len(m.Topics))
	emsd.DeleteExistingTopic(topicName)
	atomic.StoreInt32(&emsd.isLoading, 0)

	body := make([]byte, 256)
	topic := emsd.GetTopic(topicName)
	for i := 0; i < iterations; i++ {
		msg := NewMessage(topic.GenerateID(), body)
		topic.PutMessage(msg)
	}

	t.Logf("pulling from channel")
	channel1 := topic.GetChannel("ch1")

	t.Logf("read %d msgs", iterations/2)
	for i := 0; i < iterations/2; i++ {
		select {
		case msg = <-channel1.memoryMsgChan:
		case b := <-channel1.backend.ReadChan():
			msg, _ = decodeMessage(b)
		}
		t.Logf("read message %d", i+1)
		test.Equal(t, body, msg.Body)
	}

	for {
		if channel1.Depth() == int64(iterations/2) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// make sure metadata shows the topic
	m, err = getMetadata(emsd)
	test.Nil(t, err)
	test.Equal(t, 1, len(m.Topics))
	test.Equal(t, topicName, m.Topics[0].Name)

	exitChan <- 1
	<-doneExitChan

	// start up a new emsd w/ the same folder

	opts = NewOptions()
	opts.Logger = test.NewTestLogger(t)
	opts.MemQueueSize = 100
	opts.MaxBytesPerFile = 10240
	opts.DataPath = origDataPath
	_, _, emsd = mustStartEMSD(opts)

	go func() {
		<-exitChan
		emsd.Exit()
		doneExitChan <- 1
	}()

	topic = emsd.GetTopic(topicName)
	// should be empty; channel should have drained everything
	count := topic.Depth()
	test.Equal(t, int64(0), count)

	channel1 = topic.GetChannel("ch1")

	for {
		if channel1.Depth() == int64(iterations/2) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// read the other half of the messages
	for i := 0; i < iterations/2; i++ {
		select {
		case msg = <-channel1.memoryMsgChan:
		case b := <-channel1.backend.ReadChan():
			msg, _ = decodeMessage(b)
		}
		t.Logf("read message %d", i+1)
		test.Equal(t, body, msg.Body)
	}

	// verify we drained things
	test.Equal(t, 0, len(topic.memoryMsgChan))
	test.Equal(t, int64(0), topic.backend.Depth())

	exitChan <- 1
	<-doneExitChan
}

func TestEphemeralTopicsAndChannels(t *testing.T) {
	// ephemeral topics/channels are lazily removed after the last channel/client is removed
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	opts.MemQueueSize = 100
	_, _, emsd := mustStartEMSD(opts)
	defer os.RemoveAll(opts.DataPath)

	topicName := "ephemeral_topic" + strconv.Itoa(int(time.Now().Unix())) + "#ephemeral"
	doneExitChan := make(chan int)

	exitChan := make(chan int)
	go func() {
		<-exitChan
		emsd.Exit()
		doneExitChan <- 1
	}()

	body := []byte("an_ephemeral_message")
	topic := emsd.GetTopic(topicName)
	ephemeralChannel := topic.GetChannel("ch1#ephemeral")
	client := newClientV2(0, nil, emsd)
	err := ephemeralChannel.AddClient(client.ID, client)
	test.Equal(t, err, nil)

	msg := NewMessage(topic.GenerateID(), body)
	topic.PutMessage(msg)
	msg = <-ephemeralChannel.memoryMsgChan
	test.Equal(t, body, msg.Body)

	ephemeralChannel.RemoveClient(client.ID)

	time.Sleep(100 * time.Millisecond)

	topic.Lock()
	numChannels := len(topic.channelMap)
	topic.Unlock()
	test.Equal(t, 0, numChannels)

	emsd.Lock()
	numTopics := len(emsd.topicMap)
	emsd.Unlock()
	test.Equal(t, 0, numTopics)

	exitChan <- 1
	<-doneExitChan
}

func TestPauseMetadata(t *testing.T) {
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	_, _, emsd := mustStartEMSD(opts)
	defer os.RemoveAll(opts.DataPath)
	defer emsd.Exit()

	// avoid concurrency issue of async PersistMetadata() calls
	atomic.StoreInt32(&emsd.isLoading, 1)
	topicName := "pause_metadata" + strconv.Itoa(int(time.Now().Unix()))
	topic := emsd.GetTopic(topicName)
	channel := topic.GetChannel("ch")
	atomic.StoreInt32(&emsd.isLoading, 0)
	emsd.PersistMetadata()

	var isPaused = func(n *EMSD, topicIndex int, channelIndex int) bool {
		m, _ := getMetadata(n)
		return m.Topics[topicIndex].Channels[channelIndex].Paused
	}

	test.Equal(t, false, isPaused(emsd, 0, 0))

	channel.Pause()
	test.Equal(t, false, isPaused(emsd, 0, 0))

	emsd.PersistMetadata()
	test.Equal(t, true, isPaused(emsd, 0, 0))

	channel.UnPause()
	test.Equal(t, true, isPaused(emsd, 0, 0))

	emsd.PersistMetadata()
	test.Equal(t, false, isPaused(emsd, 0, 0))
}

func mustStartEMSLookupd(opts *emslookupd.Options) (*net.TCPAddr, *net.TCPAddr, *emslookupd.EMSLookupd) {
	opts.TCPAddress = "127.0.0.1:0"
	opts.HTTPAddress = "127.0.0.1:0"
	lookupd, err := emslookupd.New(opts)
	if err != nil {
		panic(err)
	}
	go func() {
		err := lookupd.Main()
		if err != nil {
			panic(err)
		}
	}()
	return lookupd.RealTCPAddr(), lookupd.RealHTTPAddr(), lookupd
}

func TestReconfigure(t *testing.T) {
	lopts := emslookupd.NewOptions()
	lopts.Logger = test.NewTestLogger(t)

	lopts1 := *lopts
	_, _, lookupd1 := mustStartEMSLookupd(&lopts1)
	defer lookupd1.Exit()
	lopts2 := *lopts
	_, _, lookupd2 := mustStartEMSLookupd(&lopts2)
	defer lookupd2.Exit()
	lopts3 := *lopts
	_, _, lookupd3 := mustStartEMSLookupd(&lopts3)
	defer lookupd3.Exit()

	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	_, _, emsd := mustStartEMSD(opts)
	defer os.RemoveAll(opts.DataPath)
	defer emsd.Exit()

	newOpts := NewOptions()
	newOpts.Logger = opts.Logger
	newOpts.EMSLookupdTCPAddresses = []string{lookupd1.RealTCPAddr().String()}
	emsd.swapOpts(newOpts)
	emsd.triggerOptsNotification()
	test.Equal(t, 1, len(emsd.getOpts().EMSLookupdTCPAddresses))

	var numLookupPeers int
	for i := 0; i < 100; i++ {
		numLookupPeers = len(emsd.lookupPeers.Load().([]*lookupPeer))
		if numLookupPeers == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	test.Equal(t, 1, numLookupPeers)

	newOpts = NewOptions()
	newOpts.Logger = opts.Logger
	newOpts.EMSLookupdTCPAddresses = []string{lookupd2.RealTCPAddr().String(), lookupd3.RealTCPAddr().String()}
	emsd.swapOpts(newOpts)
	emsd.triggerOptsNotification()
	test.Equal(t, 2, len(emsd.getOpts().EMSLookupdTCPAddresses))

	for i := 0; i < 100; i++ {
		numLookupPeers = len(emsd.lookupPeers.Load().([]*lookupPeer))
		if numLookupPeers == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	test.Equal(t, 2, numLookupPeers)

	var lookupPeers []string
	for _, lp := range emsd.lookupPeers.Load().([]*lookupPeer) {
		lookupPeers = append(lookupPeers, lp.addr)
	}
	test.Equal(t, newOpts.EMSLookupdTCPAddresses, lookupPeers)
}

func TestCluster(t *testing.T) {
	lopts := emslookupd.NewOptions()
	lopts.Logger = test.NewTestLogger(t)
	lopts.BroadcastAddress = "127.0.0.1"
	_, _, lookupd := mustStartEMSLookupd(lopts)

	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	opts.EMSLookupdTCPAddresses = []string{lookupd.RealTCPAddr().String()}
	opts.BroadcastAddress = "127.0.0.1"
	_, _, emsd := mustStartEMSD(opts)
	defer os.RemoveAll(opts.DataPath)
	defer emsd.Exit()

	topicName := "cluster_test" + strconv.Itoa(int(time.Now().Unix()))

	hostname, err := os.Hostname()
	test.Nil(t, err)

	url := fmt.Sprintf("http://%s/topic/create?topic=%s", emsd.RealHTTPAddr(), topicName)
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).POSTV1(url)
	test.Nil(t, err)

	url = fmt.Sprintf("http://%s/channel/create?topic=%s&channel=ch", emsd.RealHTTPAddr(), topicName)
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).POSTV1(url)
	test.Nil(t, err)

	// allow some time for emsd to push info to emslookupd
	time.Sleep(350 * time.Millisecond)

	var d map[string][]struct {
		Hostname         string `json:"hostname"`
		BroadcastAddress string `json:"broadcast_address"`
		TCPPort          int    `json:"tcp_port"`
		Tombstoned       bool   `json:"tombstoned"`
	}

	endpoint := fmt.Sprintf("http://%s/debug", lookupd.RealHTTPAddr())
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).GETV1(endpoint, &d)
	test.Nil(t, err)

	topicData := d["topic:"+topicName+":"]
	test.Equal(t, 1, len(topicData))

	test.Equal(t, hostname, topicData[0].Hostname)
	test.Equal(t, "127.0.0.1", topicData[0].BroadcastAddress)
	test.Equal(t, emsd.RealTCPAddr().Port, topicData[0].TCPPort)
	test.Equal(t, false, topicData[0].Tombstoned)

	channelData := d["channel:"+topicName+":ch"]
	test.Equal(t, 1, len(channelData))

	test.Equal(t, hostname, channelData[0].Hostname)
	test.Equal(t, "127.0.0.1", channelData[0].BroadcastAddress)
	test.Equal(t, emsd.RealTCPAddr().Port, channelData[0].TCPPort)
	test.Equal(t, false, channelData[0].Tombstoned)

	var lr struct {
		Producers []struct {
			Hostname         string `json:"hostname"`
			BroadcastAddress string `json:"broadcast_address"`
			TCPPort          int    `json:"tcp_port"`
		} `json:"producers"`
		Channels []string `json:"channels"`
	}

	endpoint = fmt.Sprintf("http://%s/lookup?topic=%s", lookupd.RealHTTPAddr(), topicName)
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).GETV1(endpoint, &lr)
	test.Nil(t, err)

	test.Equal(t, 1, len(lr.Producers))
	test.Equal(t, hostname, lr.Producers[0].Hostname)
	test.Equal(t, "127.0.0.1", lr.Producers[0].BroadcastAddress)
	test.Equal(t, emsd.RealTCPAddr().Port, lr.Producers[0].TCPPort)
	test.Equal(t, 1, len(lr.Channels))
	test.Equal(t, "ch", lr.Channels[0])

	url = fmt.Sprintf("http://%s/topic/delete?topic=%s", emsd.RealHTTPAddr(), topicName)
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).POSTV1(url)
	test.Nil(t, err)

	// allow some time for emsd to push info to emslookupd
	time.Sleep(350 * time.Millisecond)

	endpoint = fmt.Sprintf("http://%s/lookup?topic=%s", lookupd.RealHTTPAddr(), topicName)
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).GETV1(endpoint, &lr)
	test.Nil(t, err)

	test.Equal(t, 0, len(lr.Producers))

	var dd map[string][]interface{}
	endpoint = fmt.Sprintf("http://%s/debug", lookupd.RealHTTPAddr())
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).GETV1(endpoint, &dd)
	test.Nil(t, err)

	test.Equal(t, 0, len(dd["topic:"+topicName+":"]))
	test.Equal(t, 0, len(dd["channel:"+topicName+":ch"]))
}

func TestSetHealth(t *testing.T) {
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	emsd, err := New(opts)
	test.Nil(t, err)
	defer emsd.Exit()

	test.Nil(t, emsd.GetError())
	test.Equal(t, true, emsd.IsHealthy())

	emsd.SetHealth(nil)
	test.Nil(t, emsd.GetError())
	test.Equal(t, true, emsd.IsHealthy())

	emsd.SetHealth(errors.New("health error"))
	test.NotNil(t, emsd.GetError())
	test.Equal(t, "NOK - health error", emsd.GetHealth())
	test.Equal(t, false, emsd.IsHealthy())

	emsd.SetHealth(nil)
	test.Nil(t, emsd.GetError())
	test.Equal(t, "OK", emsd.GetHealth())
	test.Equal(t, true, emsd.IsHealthy())
}
