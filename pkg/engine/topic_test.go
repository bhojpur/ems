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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/bhojpur/ems/pkg/core/test"
)

func TestGetTopic(t *testing.T) {
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	_, _, emsd := mustStartEMSD(opts)
	defer os.RemoveAll(opts.DataPath)
	defer emsd.Exit()

	topic1 := emsd.GetTopic("test")
	test.NotNil(t, topic1)
	test.Equal(t, "test", topic1.name)

	topic2 := emsd.GetTopic("test")
	test.Equal(t, topic1, topic2)

	topic3 := emsd.GetTopic("test2")
	test.Equal(t, "test2", topic3.name)
	test.NotEqual(t, topic2, topic3)
}

func TestGetChannel(t *testing.T) {
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	_, _, emsd := mustStartEMSD(opts)
	defer os.RemoveAll(opts.DataPath)
	defer emsd.Exit()

	topic := emsd.GetTopic("test")

	channel1 := topic.GetChannel("ch1")
	test.NotNil(t, channel1)
	test.Equal(t, "ch1", channel1.name)

	channel2 := topic.GetChannel("ch2")

	test.Equal(t, channel1, topic.channelMap["ch1"])
	test.Equal(t, channel2, topic.channelMap["ch2"])
}

type errorBackendQueue struct{}

func (d *errorBackendQueue) Put([]byte) error        { return errors.New("never gonna happen") }
func (d *errorBackendQueue) ReadChan() <-chan []byte { return nil }
func (d *errorBackendQueue) Close() error            { return nil }
func (d *errorBackendQueue) Delete() error           { return nil }
func (d *errorBackendQueue) Depth() int64            { return 0 }
func (d *errorBackendQueue) Empty() error            { return nil }

type errorRecoveredBackendQueue struct{ errorBackendQueue }

func (d *errorRecoveredBackendQueue) Put([]byte) error { return nil }

func TestHealth(t *testing.T) {
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	opts.MemQueueSize = 2
	_, httpAddr, emsd := mustStartEMSD(opts)
	defer os.RemoveAll(opts.DataPath)
	defer emsd.Exit()

	topic := emsd.GetTopic("test")
	topic.backend = &errorBackendQueue{}

	msg := NewMessage(topic.GenerateID(), make([]byte, 100))
	err := topic.PutMessage(msg)
	test.Nil(t, err)

	msg = NewMessage(topic.GenerateID(), make([]byte, 100))
	err = topic.PutMessages([]*Message{msg})
	test.Nil(t, err)

	msg = NewMessage(topic.GenerateID(), make([]byte, 100))
	err = topic.PutMessage(msg)
	test.NotNil(t, err)

	msg = NewMessage(topic.GenerateID(), make([]byte, 100))
	err = topic.PutMessages([]*Message{msg})
	test.NotNil(t, err)

	url := fmt.Sprintf("http://%s/ping", httpAddr)
	resp, err := http.Get(url)
	test.Nil(t, err)
	test.Equal(t, 500, resp.StatusCode)
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	test.Equal(t, "NOK - never gonna happen", string(body))

	topic.backend = &errorRecoveredBackendQueue{}

	msg = NewMessage(topic.GenerateID(), make([]byte, 100))
	err = topic.PutMessages([]*Message{msg})
	test.Nil(t, err)

	resp, err = http.Get(url)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	test.Equal(t, "OK", string(body))
}

func TestDeletes(t *testing.T) {
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	_, _, emsd := mustStartEMSD(opts)
	defer os.RemoveAll(opts.DataPath)
	defer emsd.Exit()

	topic := emsd.GetTopic("test")

	channel1 := topic.GetChannel("ch1")
	test.NotNil(t, channel1)

	err := topic.DeleteExistingChannel("ch1")
	test.Nil(t, err)
	test.Equal(t, 0, len(topic.channelMap))

	channel2 := topic.GetChannel("ch2")
	test.NotNil(t, channel2)

	err = emsd.DeleteExistingTopic("test")
	test.Nil(t, err)
	test.Equal(t, 0, len(topic.channelMap))
	test.Equal(t, 0, len(emsd.topicMap))
}

func TestDeleteLast(t *testing.T) {
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	_, _, emsd := mustStartEMSD(opts)
	defer os.RemoveAll(opts.DataPath)
	defer emsd.Exit()

	topic := emsd.GetTopic("test")

	channel1 := topic.GetChannel("ch1")
	test.NotNil(t, channel1)

	err := topic.DeleteExistingChannel("ch1")
	test.Nil(t, err)
	test.Equal(t, 0, len(topic.channelMap))

	msg := NewMessage(topic.GenerateID(), []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	err = topic.PutMessage(msg)
	time.Sleep(100 * time.Millisecond)
	test.Nil(t, err)
	test.Equal(t, int64(1), topic.Depth())
}

func TestPause(t *testing.T) {
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(t)
	_, _, emsd := mustStartEMSD(opts)
	defer os.RemoveAll(opts.DataPath)
	defer emsd.Exit()

	topicName := "test_topic_pause" + strconv.Itoa(int(time.Now().Unix()))
	topic := emsd.GetTopic(topicName)
	err := topic.Pause()
	test.Nil(t, err)

	channel := topic.GetChannel("ch1")
	test.NotNil(t, channel)

	msg := NewMessage(topic.GenerateID(), []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	err = topic.PutMessage(msg)
	test.Nil(t, err)

	time.Sleep(15 * time.Millisecond)

	test.Equal(t, int64(1), topic.Depth())
	test.Equal(t, int64(0), channel.Depth())

	err = topic.UnPause()
	test.Nil(t, err)

	time.Sleep(15 * time.Millisecond)

	test.Equal(t, int64(0), topic.Depth())
	test.Equal(t, int64(1), channel.Depth())
}

func BenchmarkTopicPut(b *testing.B) {
	b.StopTimer()
	topicName := "bench_topic_put" + strconv.Itoa(b.N)
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(b)
	opts.MemQueueSize = int64(b.N)
	_, _, emsd := mustStartEMSD(opts)
	defer os.RemoveAll(opts.DataPath)
	defer emsd.Exit()
	b.StartTimer()

	for i := 0; i <= b.N; i++ {
		topic := emsd.GetTopic(topicName)
		msg := NewMessage(topic.GenerateID(), []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaa"))
		topic.PutMessage(msg)
	}
}

func BenchmarkTopicToChannelPut(b *testing.B) {
	b.StopTimer()
	topicName := "bench_topic_to_channel_put" + strconv.Itoa(b.N)
	channelName := "bench"
	opts := NewOptions()
	opts.Logger = test.NewTestLogger(b)
	opts.MemQueueSize = int64(b.N)
	_, _, emsd := mustStartEMSD(opts)
	defer os.RemoveAll(opts.DataPath)
	defer emsd.Exit()
	channel := emsd.GetTopic(topicName).GetChannel(channelName)
	b.StartTimer()

	for i := 0; i <= b.N; i++ {
		topic := emsd.GetTopic(topicName)
		msg := NewMessage(topic.GenerateID(), []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaa"))
		topic.PutMessage(msg)
	}

	for {
		if len(channel.memoryMsgChan) == b.N {
			break
		}
		runtime.Gosched()
	}
}
