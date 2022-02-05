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
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/bhojpur/ems/pkg/core/http_api"
	"github.com/bhojpur/ems/pkg/core/test"
	"github.com/golang/snappy"
)

func TestStats(t *testing.T) {
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	tcpAddr, _, emsd := mustStartEMSD(opts)
	defer os.RemoveAll(opts.DataPath)
	defer emsd.Exit()

	topicName := "test_stats" + strconv.Itoa(int(time.Now().Unix()))
	topic := emsd.GetTopic(topicName)
	msg := NewMessage(topic.GenerateID(), []byte("test body"))
	topic.PutMessage(msg)

	accompanyTopicName := "accompany_test_stats" + strconv.Itoa(int(time.Now().Unix()))
	accompanyTopic := emsd.GetTopic(accompanyTopicName)
	msg = NewMessage(accompanyTopic.GenerateID(), []byte("accompany test body"))
	accompanyTopic.PutMessage(msg)

	conn, err := mustConnectEMSD(tcpAddr)
	test.Nil(t, err)
	defer conn.Close()

	identify(t, conn, nil, frameTypeResponse)
	sub(t, conn, topicName, "ch")

	stats := emsd.GetStats(topicName, "ch", true).Topics
	t.Logf("stats: %+v", stats)

	test.Equal(t, 1, len(stats))
	test.Equal(t, 1, len(stats[0].Channels))
	test.Equal(t, 1, len(stats[0].Channels[0].Clients))
	test.Equal(t, 1, stats[0].Channels[0].ClientCount)

	stats = emsd.GetStats(topicName, "ch", false).Topics
	t.Logf("stats: %+v", stats)

	test.Equal(t, 1, len(stats))
	test.Equal(t, 1, len(stats[0].Channels))
	test.Equal(t, 0, len(stats[0].Channels[0].Clients))
	test.Equal(t, 1, stats[0].Channels[0].ClientCount)

	stats = emsd.GetStats(topicName, "none_exist_channel", false).Topics
	t.Logf("stats: %+v", stats)

	test.Equal(t, 0, len(stats))

	stats = emsd.GetStats("none_exist_topic", "none_exist_channel", false).Topics
	t.Logf("stats: %+v", stats)

	test.Equal(t, 0, len(stats))
}

func TestClientAttributes(t *testing.T) {
	userAgent := "Test User Agent"

	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	opts.LogLevel = LOG_DEBUG
	opts.SnappyEnabled = true
	tcpAddr, httpAddr, emsd := mustStartEMSD(opts)
	defer os.RemoveAll(opts.DataPath)
	defer emsd.Exit()

	conn, err := mustConnectEMSD(tcpAddr)
	test.Nil(t, err)
	defer conn.Close()

	data := identify(t, conn, map[string]interface{}{
		"snappy":     true,
		"user_agent": userAgent,
	}, frameTypeResponse)
	resp := struct {
		Snappy    bool   `json:"snappy"`
		UserAgent string `json:"user_agent"`
	}{}
	err = json.Unmarshal(data, &resp)
	test.Nil(t, err)
	test.Equal(t, true, resp.Snappy)

	r := snappy.NewReader(conn)
	w := snappy.NewWriter(conn)
	readValidate(t, r, frameTypeResponse, "OK")

	topicName := "test_client_attributes" + strconv.Itoa(int(time.Now().Unix()))
	sub(t, readWriter{r, w}, topicName, "ch")

	var d struct {
		Topics []struct {
			Channels []struct {
				Clients []struct {
					UserAgent string `json:"user_agent"`
					Snappy    bool   `json:"snappy"`
				} `json:"clients"`
			} `json:"channels"`
		} `json:"topics"`
	}

	endpoint := fmt.Sprintf("http://127.0.0.1:%d/stats?format=json", httpAddr.Port)
	err = http_api.NewClient(nil, ConnectTimeout, RequestTimeout).GETV1(endpoint, &d)
	test.Nil(t, err)

	test.Equal(t, userAgent, d.Topics[0].Channels[0].Clients[0].UserAgent)
	test.Equal(t, true, d.Topics[0].Channels[0].Clients[0].Snappy)
}

func TestStatsChannelLocking(t *testing.T) {
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	_, _, emsd := mustStartEMSD(opts)
	defer os.RemoveAll(opts.DataPath)
	defer emsd.Exit()

	topicName := "test_channel_empty" + strconv.Itoa(int(time.Now().Unix()))
	topic := emsd.GetTopic(topicName)
	channel := topic.GetChannel("channel")

	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		for i := 0; i < 25; i++ {
			msg := NewMessage(topic.GenerateID(), []byte("test"))
			topic.PutMessage(msg)
			channel.StartInFlightTimeout(msg, 0, opts.MsgTimeout)
		}
		wg.Done()
	}()

	go func() {
		for i := 0; i < 25; i++ {
			emsd.GetStats("", "", true)
		}
		wg.Done()
	}()

	wg.Wait()

	stats := emsd.GetStats(topicName, "channel", false).Topics
	t.Logf("stats: %+v", stats)

	test.Equal(t, 1, len(stats))
	test.Equal(t, 1, len(stats[0].Channels))
	test.Equal(t, 25, stats[0].Channels[0].InFlightCount)
}
