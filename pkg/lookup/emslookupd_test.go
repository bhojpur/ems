package lookup

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
	"fmt"
	"net"
	"testing"
	"time"

	emsctl "github.com/bhojpur/ems/pkg/client"
	"github.com/bhojpur/ems/pkg/core/clusterinfo"
	"github.com/bhojpur/ems/pkg/core/http_api"
	"github.com/bhojpur/ems/pkg/core/test"
)

const (
	ConnectTimeout = 2 * time.Second
	RequestTimeout = 5 * time.Second
	TCPPort        = 5000
	HTTPPort       = 5555
	HostAddr       = "ip.address"
	EMSDVersion    = "fake-version"
)

type ProducersDoc struct {
	Producers []interface{} `json:"producers"`
}

type TopicsDoc struct {
	Topics []interface{} `json:"topics"`
}

type LookupDoc struct {
	Channels  []interface{} `json:"channels"`
	Producers []*PeerInfo   `json:"producers"`
}

func mustStartLookupd(opts *Options) (*net.TCPAddr, *net.TCPAddr, *EMSLookupd) {
	opts.TCPAddress = "127.0.0.1:0"
	opts.HTTPAddress = "127.0.0.1:0"
	emslookupd, err := New(opts)
	if err != nil {
		panic(err)
	}
	go func() {
		err := emslookupd.Main()
		if err != nil {
			panic(err)
		}
	}()
	return emslookupd.RealTCPAddr(), emslookupd.RealHTTPAddr(), emslookupd
}

func mustConnectLookupd(t *testing.T, tcpAddr *net.TCPAddr) net.Conn {
	conn, err := net.DialTimeout("tcp", tcpAddr.String(), time.Second)
	if err != nil {
		t.Fatal("failed to connect to lookupd")
	}
	conn.Write(emsctl.MagicV1)
	return conn
}

func identify(t *testing.T, conn net.Conn) {
	ci := make(map[string]interface{})
	ci["tcp_port"] = TCPPort
	ci["http_port"] = HTTPPort
	ci["broadcast_address"] = HostAddr
	ci["hostname"] = HostAddr
	ci["version"] = EMSDVersion
	cmd, _ := emsctl.Identify(ci)
	_, err := cmd.WriteTo(conn)
	test.Nil(t, err)
	_, err = emsctl.ReadResponse(conn)
	test.Nil(t, err)
}

func TestBasicLookupd(t *testing.T) {
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	tcpAddr, httpAddr, emslookupd := mustStartLookupd(opts)
	defer emslookupd.Exit()

	topics := emslookupd.DB.FindRegistrations("topic", "*", "*")
	test.Equal(t, 0, len(topics))

	topicName := "connectmsg"

	conn := mustConnectLookupd(t, tcpAddr)

	identify(t, conn)

	emsctl.Register(topicName, "channel1").WriteTo(conn)
	v, err := emsctl.ReadResponse(conn)
	test.Nil(t, err)
	test.Equal(t, []byte("OK"), v)

	pr := ProducersDoc{}
	endpoint := fmt.Sprintf("http://%s/nodes", httpAddr)
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).GETV1(endpoint, &pr)
	test.Nil(t, err)

	t.Logf("got %v", pr)
	test.Equal(t, 1, len(pr.Producers))

	topics = emslookupd.DB.FindRegistrations("topic", topicName, "")
	test.Equal(t, 1, len(topics))

	producers := emslookupd.DB.FindProducers("topic", topicName, "")
	test.Equal(t, 1, len(producers))
	producer := producers[0]

	test.Equal(t, HostAddr, producer.peerInfo.BroadcastAddress)
	test.Equal(t, HostAddr, producer.peerInfo.Hostname)
	test.Equal(t, TCPPort, producer.peerInfo.TCPPort)
	test.Equal(t, HTTPPort, producer.peerInfo.HTTPPort)

	tr := TopicsDoc{}
	endpoint = fmt.Sprintf("http://%s/topics", httpAddr)
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).GETV1(endpoint, &tr)
	test.Nil(t, err)

	t.Logf("got %v", tr)
	test.Equal(t, 1, len(tr.Topics))

	lr := LookupDoc{}
	endpoint = fmt.Sprintf("http://%s/lookup?topic=%s", httpAddr, topicName)
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).GETV1(endpoint, &lr)
	test.Nil(t, err)

	t.Logf("got %v", lr)
	test.Equal(t, 1, len(lr.Channels))
	test.Equal(t, 1, len(lr.Producers))
	for _, p := range lr.Producers {
		test.Equal(t, TCPPort, p.TCPPort)
		test.Equal(t, HTTPPort, p.HTTPPort)
		test.Equal(t, HostAddr, p.BroadcastAddress)
		test.Equal(t, EMSDVersion, p.Version)
	}

	conn.Close()
	time.Sleep(10 * time.Millisecond)

	// now there should be no producers, but still topic/channel entries
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).GETV1(endpoint, &lr)
	test.Nil(t, err)

	test.Equal(t, 1, len(lr.Channels))
	test.Equal(t, 0, len(lr.Producers))
}

func TestChannelUnregister(t *testing.T) {
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	tcpAddr, httpAddr, emslookupd := mustStartLookupd(opts)
	defer emslookupd.Exit()

	topics := emslookupd.DB.FindRegistrations("topic", "*", "*")
	test.Equal(t, 0, len(topics))

	topicName := "channel_unregister"

	conn := mustConnectLookupd(t, tcpAddr)
	defer conn.Close()

	identify(t, conn)

	emsctl.Register(topicName, "ch1").WriteTo(conn)
	v, err := emsctl.ReadResponse(conn)
	test.Nil(t, err)
	test.Equal(t, []byte("OK"), v)

	topics = emslookupd.DB.FindRegistrations("topic", topicName, "")
	test.Equal(t, 1, len(topics))

	channels := emslookupd.DB.FindRegistrations("channel", topicName, "*")
	test.Equal(t, 1, len(channels))

	emsctl.UnRegister(topicName, "ch1").WriteTo(conn)
	v, err = emsctl.ReadResponse(conn)
	test.Nil(t, err)
	test.Equal(t, []byte("OK"), v)

	topics = emslookupd.DB.FindRegistrations("topic", topicName, "")
	test.Equal(t, 1, len(topics))

	// we should still have mention of the topic even though there is no producer
	// (ie. we haven't *deleted* the channel, just unregistered as a producer)
	channels = emslookupd.DB.FindRegistrations("channel", topicName, "*")
	test.Equal(t, 1, len(channels))

	pr := ProducersDoc{}
	endpoint := fmt.Sprintf("http://%s/lookup?topic=%s", httpAddr, topicName)
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).GETV1(endpoint, &pr)
	test.Nil(t, err)
	t.Logf("got %v", pr)
	test.Equal(t, 1, len(pr.Producers))
}

func TestTombstoneRecover(t *testing.T) {
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	opts.TombstoneLifetime = 50 * time.Millisecond
	tcpAddr, httpAddr, emslookupd := mustStartLookupd(opts)
	defer emslookupd.Exit()

	topicName := "tombstone_recover"
	topicName2 := topicName + "2"

	conn := mustConnectLookupd(t, tcpAddr)
	defer conn.Close()

	identify(t, conn)

	emsctl.Register(topicName, "channel1").WriteTo(conn)
	_, err := emsctl.ReadResponse(conn)
	test.Nil(t, err)

	emsctl.Register(topicName2, "channel2").WriteTo(conn)
	_, err = emsctl.ReadResponse(conn)
	test.Nil(t, err)

	endpoint := fmt.Sprintf("http://%s/topic/tombstone?topic=%s&node=%s:%d",
		httpAddr, topicName, HostAddr, HTTPPort)
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).POSTV1(endpoint)
	test.Nil(t, err)

	pr := ProducersDoc{}

	endpoint = fmt.Sprintf("http://%s/lookup?topic=%s", httpAddr, topicName)
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).GETV1(endpoint, &pr)
	test.Nil(t, err)
	test.Equal(t, 0, len(pr.Producers))

	endpoint = fmt.Sprintf("http://%s/lookup?topic=%s", httpAddr, topicName2)
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).GETV1(endpoint, &pr)
	test.Nil(t, err)
	test.Equal(t, 1, len(pr.Producers))

	time.Sleep(75 * time.Millisecond)

	endpoint = fmt.Sprintf("http://%s/lookup?topic=%s", httpAddr, topicName)
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).GETV1(endpoint, &pr)
	test.Nil(t, err)
	test.Equal(t, 1, len(pr.Producers))
}

func TestTombstoneUnregister(t *testing.T) {
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	opts.TombstoneLifetime = 50 * time.Millisecond
	tcpAddr, httpAddr, emslookupd := mustStartLookupd(opts)
	defer emslookupd.Exit()

	topicName := "tombstone_unregister"

	conn := mustConnectLookupd(t, tcpAddr)
	defer conn.Close()

	identify(t, conn)

	emsctl.Register(topicName, "channel1").WriteTo(conn)
	_, err := emsctl.ReadResponse(conn)
	test.Nil(t, err)

	endpoint := fmt.Sprintf("http://%s/topic/tombstone?topic=%s&node=%s:%d",
		httpAddr, topicName, HostAddr, HTTPPort)
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).POSTV1(endpoint)
	test.Nil(t, err)

	pr := ProducersDoc{}

	endpoint = fmt.Sprintf("http://%s/lookup?topic=%s", httpAddr, topicName)
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).GETV1(endpoint, &pr)
	test.Nil(t, err)
	test.Equal(t, 0, len(pr.Producers))

	emsctl.UnRegister(topicName, "").WriteTo(conn)
	_, err = emsctl.ReadResponse(conn)
	test.Nil(t, err)

	time.Sleep(55 * time.Millisecond)

	endpoint = fmt.Sprintf("http://%s/lookup?topic=%s", httpAddr, topicName)
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).GETV1(endpoint, &pr)
	test.Nil(t, err)
	test.Equal(t, 0, len(pr.Producers))
}

func TestInactiveNodes(t *testing.T) {
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	opts.InactiveProducerTimeout = 200 * time.Millisecond
	tcpAddr, httpAddr, emslookupd := mustStartLookupd(opts)
	defer emslookupd.Exit()

	lookupdHTTPAddrs := []string{fmt.Sprintf("%s", httpAddr)}

	topicName := "inactive_nodes"

	conn := mustConnectLookupd(t, tcpAddr)
	defer conn.Close()

	identify(t, conn)

	emsctl.Register(topicName, "channel1").WriteTo(conn)
	_, err := emsctl.ReadResponse(conn)
	test.Nil(t, err)

	ci := clusterinfo.New(nil, http_api.NewClient(nil, ConnectTimeout, RequestTimeout))

	producers, _ := ci.GetLookupdProducers(lookupdHTTPAddrs)
	test.Equal(t, 1, len(producers))
	test.Equal(t, 1, len(producers[0].Topics))
	test.Equal(t, topicName, producers[0].Topics[0].Topic)
	test.Equal(t, false, producers[0].Topics[0].Tombstoned)

	time.Sleep(250 * time.Millisecond)

	producers, _ = ci.GetLookupdProducers(lookupdHTTPAddrs)
	test.Equal(t, 0, len(producers))
}

func TestTombstonedNodes(t *testing.T) {
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	tcpAddr, httpAddr, emslookupd := mustStartLookupd(opts)
	defer emslookupd.Exit()

	lookupdHTTPAddrs := []string{fmt.Sprintf("%s", httpAddr)}

	topicName := "inactive_nodes"

	conn := mustConnectLookupd(t, tcpAddr)
	defer conn.Close()

	identify(t, conn)

	emsctl.Register(topicName, "channel1").WriteTo(conn)
	_, err := emsctl.ReadResponse(conn)
	test.Nil(t, err)

	ci := clusterinfo.New(nil, http_api.NewClient(nil, ConnectTimeout, RequestTimeout))

	producers, _ := ci.GetLookupdProducers(lookupdHTTPAddrs)
	test.Equal(t, 1, len(producers))
	test.Equal(t, 1, len(producers[0].Topics))
	test.Equal(t, topicName, producers[0].Topics[0].Topic)
	test.Equal(t, false, producers[0].Topics[0].Tombstoned)

	endpoint := fmt.Sprintf("http://%s/topic/tombstone?topic=%s&node=%s:%d",
		httpAddr, topicName, HostAddr, HTTPPort)
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).POSTV1(endpoint)
	test.Nil(t, err)

	producers, _ = ci.GetLookupdProducers(lookupdHTTPAddrs)
	test.Equal(t, 1, len(producers))
	test.Equal(t, 1, len(producers[0].Topics))
	test.Equal(t, topicName, producers[0].Topics[0].Topic)
	test.Equal(t, true, producers[0].Topics[0].Tombstoned)
}
