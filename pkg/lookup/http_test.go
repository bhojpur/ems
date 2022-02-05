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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/bhojpur/ems/pkg/core/test"
	"github.com/bhojpur/ems/pkg/core/version"
	emssvr "github.com/bhojpur/ems/pkg/engine"
)

type InfoDoc struct {
	Version string `json:"version"`
}

type ChannelsDoc struct {
	Channels []interface{} `json:"channels"`
}

type ErrMessage struct {
	Message string `json:"message"`
}

func bootstrapEMSCluster(t *testing.T) (string, []*emssvr.EMSD, *EMSLookupd) {
	lgr := test.NewTestLogger(t)

	emslookupdOpts := NewOptions()
	emslookupdOpts.TCPAddress = "127.0.0.1:0"
	emslookupdOpts.HTTPAddress = "127.0.0.1:0"
	emslookupdOpts.BroadcastAddress = "127.0.0.1"
	emslookupdOpts.Logger = lgr
	emslookupd1, err := New(emslookupdOpts)
	if err != nil {
		panic(err)
	}
	go func() {
		err := emslookupd1.Main()
		if err != nil {
			panic(err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	emsdOpts := emssvr.NewOptions()
	emsdOpts.TCPAddress = "127.0.0.1:0"
	emsdOpts.HTTPAddress = "127.0.0.1:0"
	emsdOpts.BroadcastAddress = "127.0.0.1"
	emsdOpts.EMSLookupdTCPAddresses = []string{emslookupd1.RealTCPAddr().String()}
	emsdOpts.Logger = lgr
	tmpDir, err := ioutil.TempDir("", fmt.Sprintf("ems-test-%d", time.Now().UnixNano()))
	if err != nil {
		panic(err)
	}
	emsdOpts.DataPath = tmpDir
	emsd1, err := emssvr.New(emsdOpts)
	if err != nil {
		panic(err)
	}
	go func() {
		err := emsd1.Main()
		if err != nil {
			panic(err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	return tmpDir, []*emssvr.EMSD{emsd1}, emslookupd1
}

func makeTopic(emslookupd *EMSLookupd, topicName string) {
	key := Registration{"topic", topicName, ""}
	emslookupd.DB.AddRegistration(key)
}

func makeChannel(emslookupd *EMSLookupd, topicName string, channelName string) {
	key := Registration{"channel", topicName, channelName}
	emslookupd.DB.AddRegistration(key)
	makeTopic(emslookupd, topicName)
}

func TestPing(t *testing.T) {
	dataPath, emsds, emslookupd1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupd1.Exit()

	client := http.Client{}
	url := fmt.Sprintf("http://%s/ping", emslookupd1.RealHTTPAddr())
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	test.Equal(t, []byte("OK"), body)
}

func TestInfo(t *testing.T) {
	dataPath, emsds, emslookupd1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupd1.Exit()

	client := http.Client{}
	url := fmt.Sprintf("http://%s/info", emslookupd1.RealHTTPAddr())
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	info := InfoDoc{}
	err = json.Unmarshal(body, &info)
	test.Nil(t, err)
	test.Equal(t, version.Binary, info.Version)
}

func TestCreateTopic(t *testing.T) {
	dataPath, emsds, emslookupd1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupd1.Exit()

	em := ErrMessage{}
	client := http.Client{}
	url := fmt.Sprintf("http://%s/topic/create", emslookupd1.RealHTTPAddr())

	req, _ := http.NewRequest("POST", url, nil)
	resp, err := client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 400, resp.StatusCode)
	test.Equal(t, "Bad Request", http.StatusText(resp.StatusCode))
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	err = json.Unmarshal(body, &em)
	test.Nil(t, err)
	test.Equal(t, "MISSING_ARG_TOPIC", em.Message)

	topicName := "sampletopicA" + strconv.Itoa(int(time.Now().Unix())) + "$"
	url = fmt.Sprintf("http://%s/topic/create?topic=%s", emslookupd1.RealHTTPAddr(), topicName)

	req, _ = http.NewRequest("POST", url, nil)
	resp, err = client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 400, resp.StatusCode)
	test.Equal(t, "Bad Request", http.StatusText(resp.StatusCode))
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	err = json.Unmarshal(body, &em)
	test.Nil(t, err)
	test.Equal(t, "INVALID_ARG_TOPIC", em.Message)

	topicName = "sampletopicA" + strconv.Itoa(int(time.Now().Unix()))
	url = fmt.Sprintf("http://%s/topic/create?topic=%s", emslookupd1.RealHTTPAddr(), topicName)

	req, _ = http.NewRequest("POST", url, nil)
	resp, err = client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	test.Equal(t, []byte(""), body)
}

func TestDeleteTopic(t *testing.T) {
	dataPath, emsds, emslookupd1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupd1.Exit()

	em := ErrMessage{}
	client := http.Client{}
	url := fmt.Sprintf("http://%s/topic/delete", emslookupd1.RealHTTPAddr())

	req, _ := http.NewRequest("POST", url, nil)
	resp, err := client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 400, resp.StatusCode)
	test.Equal(t, "Bad Request", http.StatusText(resp.StatusCode))
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	err = json.Unmarshal(body, &em)
	test.Nil(t, err)
	test.Equal(t, "MISSING_ARG_TOPIC", em.Message)

	topicName := "sampletopicA" + strconv.Itoa(int(time.Now().Unix()))
	makeTopic(emslookupd1, topicName)

	url = fmt.Sprintf("http://%s/topic/delete?topic=%s", emslookupd1.RealHTTPAddr(), topicName)

	req, _ = http.NewRequest("POST", url, nil)
	resp, err = client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	test.Equal(t, []byte(""), body)

	topicName = "sampletopicB" + strconv.Itoa(int(time.Now().Unix()))
	channelName := "foobar" + strconv.Itoa(int(time.Now().Unix()))
	makeChannel(emslookupd1, topicName, channelName)

	url = fmt.Sprintf("http://%s/topic/delete?topic=%s", emslookupd1.RealHTTPAddr(), topicName)

	req, _ = http.NewRequest("POST", url, nil)
	resp, err = client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	test.Equal(t, []byte(""), body)
}

func TestGetChannels(t *testing.T) {
	dataPath, emsds, emslookupd1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupd1.Exit()

	client := http.Client{}
	url := fmt.Sprintf("http://%s/channels", emslookupd1.RealHTTPAddr())

	em := ErrMessage{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Accept", "application/vnd.ems; version=1.0")
	resp, err := client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 400, resp.StatusCode)
	test.Equal(t, "Bad Request", http.StatusText(resp.StatusCode))
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	err = json.Unmarshal(body, &em)
	test.Nil(t, err)
	test.Equal(t, "MISSING_ARG_TOPIC", em.Message)

	ch := ChannelsDoc{}
	topicName := "sampletopicA" + strconv.Itoa(int(time.Now().Unix()))
	makeTopic(emslookupd1, topicName)

	url = fmt.Sprintf("http://%s/channels?topic=%s", emslookupd1.RealHTTPAddr(), topicName)

	req, _ = http.NewRequest("GET", url, nil)
	req.Header.Add("Accept", "application/vnd.ems; version=1.0")
	resp, err = client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	err = json.Unmarshal(body, &ch)
	test.Nil(t, err)
	test.Equal(t, 0, len(ch.Channels))

	topicName = "sampletopicB" + strconv.Itoa(int(time.Now().Unix()))
	channelName := "foobar" + strconv.Itoa(int(time.Now().Unix()))
	makeChannel(emslookupd1, topicName, channelName)

	url = fmt.Sprintf("http://%s/channels?topic=%s", emslookupd1.RealHTTPAddr(), topicName)

	req, _ = http.NewRequest("GET", url, nil)
	req.Header.Add("Accept", "application/vnd.ems; version=1.0")
	resp, err = client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	err = json.Unmarshal(body, &ch)
	test.Nil(t, err)
	test.Equal(t, 1, len(ch.Channels))
	test.Equal(t, channelName, ch.Channels[0])
}

func TestCreateChannel(t *testing.T) {
	dataPath, emsds, emslookupd1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupd1.Exit()

	em := ErrMessage{}
	client := http.Client{}
	url := fmt.Sprintf("http://%s/channel/create", emslookupd1.RealHTTPAddr())

	req, _ := http.NewRequest("POST", url, nil)
	resp, err := client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 400, resp.StatusCode)
	test.Equal(t, "Bad Request", http.StatusText(resp.StatusCode))
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	err = json.Unmarshal(body, &em)
	test.Nil(t, err)
	test.Equal(t, "MISSING_ARG_TOPIC", em.Message)

	topicName := "sampletopicB" + strconv.Itoa(int(time.Now().Unix())) + "$"
	url = fmt.Sprintf("http://%s/channel/create?topic=%s", emslookupd1.RealHTTPAddr(), topicName)

	req, _ = http.NewRequest("POST", url, nil)
	resp, err = client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 400, resp.StatusCode)
	test.Equal(t, "Bad Request", http.StatusText(resp.StatusCode))
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	err = json.Unmarshal(body, &em)
	test.Nil(t, err)
	test.Equal(t, "INVALID_ARG_TOPIC", em.Message)

	topicName = "sampletopicB" + strconv.Itoa(int(time.Now().Unix()))
	url = fmt.Sprintf("http://%s/channel/create?topic=%s", emslookupd1.RealHTTPAddr(), topicName)

	req, _ = http.NewRequest("POST", url, nil)
	resp, err = client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 400, resp.StatusCode)
	test.Equal(t, "Bad Request", http.StatusText(resp.StatusCode))
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	err = json.Unmarshal(body, &em)
	test.Nil(t, err)
	test.Equal(t, "MISSING_ARG_CHANNEL", em.Message)

	channelName := "foobar" + strconv.Itoa(int(time.Now().Unix())) + "$"
	url = fmt.Sprintf("http://%s/channel/create?topic=%s&channel=%s", emslookupd1.RealHTTPAddr(), topicName, channelName)

	req, _ = http.NewRequest("POST", url, nil)
	resp, err = client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 400, resp.StatusCode)
	test.Equal(t, "Bad Request", http.StatusText(resp.StatusCode))
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	err = json.Unmarshal(body, &em)
	test.Nil(t, err)
	test.Equal(t, "INVALID_ARG_CHANNEL", em.Message)

	channelName = "foobar" + strconv.Itoa(int(time.Now().Unix()))
	url = fmt.Sprintf("http://%s/channel/create?topic=%s&channel=%s", emslookupd1.RealHTTPAddr(), topicName, channelName)

	req, _ = http.NewRequest("POST", url, nil)
	resp, err = client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	test.Equal(t, []byte(""), body)
}

func TestDeleteChannel(t *testing.T) {
	dataPath, emsds, emslookupd1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupd1.Exit()

	em := ErrMessage{}
	client := http.Client{}
	url := fmt.Sprintf("http://%s/channel/delete", emslookupd1.RealHTTPAddr())

	req, _ := http.NewRequest("POST", url, nil)
	resp, err := client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 400, resp.StatusCode)
	test.Equal(t, "Bad Request", http.StatusText(resp.StatusCode))
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	err = json.Unmarshal(body, &em)
	test.Nil(t, err)
	test.Equal(t, "MISSING_ARG_TOPIC", em.Message)

	topicName := "sampletopicB" + strconv.Itoa(int(time.Now().Unix())) + "$"
	url = fmt.Sprintf("http://%s/channel/delete?topic=%s", emslookupd1.RealHTTPAddr(), topicName)

	req, _ = http.NewRequest("POST", url, nil)
	resp, err = client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 400, resp.StatusCode)
	test.Equal(t, "Bad Request", http.StatusText(resp.StatusCode))
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	err = json.Unmarshal(body, &em)
	test.Nil(t, err)
	test.Equal(t, "INVALID_ARG_TOPIC", em.Message)

	topicName = "sampletopicB" + strconv.Itoa(int(time.Now().Unix()))
	url = fmt.Sprintf("http://%s/channel/delete?topic=%s", emslookupd1.RealHTTPAddr(), topicName)

	req, _ = http.NewRequest("POST", url, nil)
	resp, err = client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 400, resp.StatusCode)
	test.Equal(t, "Bad Request", http.StatusText(resp.StatusCode))
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	err = json.Unmarshal(body, &em)
	test.Nil(t, err)
	test.Equal(t, "MISSING_ARG_CHANNEL", em.Message)

	channelName := "foobar" + strconv.Itoa(int(time.Now().Unix())) + "$"
	url = fmt.Sprintf("http://%s/channel/delete?topic=%s&channel=%s", emslookupd1.RealHTTPAddr(), topicName, channelName)

	req, _ = http.NewRequest("POST", url, nil)
	resp, err = client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 400, resp.StatusCode)
	test.Equal(t, "Bad Request", http.StatusText(resp.StatusCode))
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	err = json.Unmarshal(body, &em)
	test.Nil(t, err)
	test.Equal(t, "INVALID_ARG_CHANNEL", em.Message)

	channelName = "foobar" + strconv.Itoa(int(time.Now().Unix()))
	url = fmt.Sprintf("http://%s/channel/delete?topic=%s&channel=%s", emslookupd1.RealHTTPAddr(), topicName, channelName)

	req, _ = http.NewRequest("POST", url, nil)
	resp, err = client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 404, resp.StatusCode)
	test.Equal(t, "Not Found", http.StatusText(resp.StatusCode))
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	err = json.Unmarshal(body, &em)
	test.Nil(t, err)
	test.Equal(t, "CHANNEL_NOT_FOUND", em.Message)

	makeChannel(emslookupd1, topicName, channelName)

	req, _ = http.NewRequest("POST", url, nil)
	resp, err = client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	test.Equal(t, []byte(""), body)
}
