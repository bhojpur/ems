package admin

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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/bhojpur/ems/pkg/core/clusterinfo"
	"github.com/bhojpur/ems/pkg/core/test"
	"github.com/bhojpur/ems/pkg/core/version"
	emssvr "github.com/bhojpur/ems/pkg/engine"
	emslookupd "github.com/bhojpur/ems/pkg/lookup"
)

type TopicsDoc struct {
	Topics []interface{} `json:"topics"`
}

type TopicStatsDoc struct {
	*clusterinfo.TopicStats
	Message string `json:"message"`
}

type NodesDoc struct {
	Nodes   clusterinfo.Producers `json:"nodes"`
	Message string                `json:"message"`
}

type NodeStatsDoc struct {
	Node          string                    `json:"node"`
	TopicStats    []*clusterinfo.TopicStats `json:"topics"`
	TotalMessages int64                     `json:"total_messages"`
	TotalClients  int64                     `json:"total_clients"`
	Message       string                    `json:"message"`
}

type ChannelStatsDoc struct {
	*clusterinfo.ChannelStats
	Message string `json:"message"`
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

func bootstrapEMSCluster(t *testing.T) (string, []*emssvr.EMSD, []*emslookupd.EMSLookupd, *EMSAdmin) {
	return bootstrapEMSClusterWithAuth(t, false)
}

func bootstrapEMSClusterWithAuth(t *testing.T, withAuth bool) (string, []*emssvr.EMSD, []*emslookupd.EMSLookupd, *EMSAdmin) {
	lgr := test.NewTestLogger(t)

	emslookupdOpts := emslookupd.NewOptions()
	emslookupdOpts.TCPAddress = "127.0.0.1:0"
	emslookupdOpts.HTTPAddress = "127.0.0.1:0"
	emslookupdOpts.BroadcastAddress = "127.0.0.1"
	emslookupdOpts.Logger = lgr
	emslookupd1, err := emslookupd.New(emslookupdOpts)
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
	tmpDir, err := ioutil.TempDir("", "ems-test-")
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

	emsadminOpts := NewOptions()
	emsadminOpts.HTTPAddress = "127.0.0.1:0"
	emsadminOpts.EMSLookupdHTTPAddresses = []string{emslookupd1.RealHTTPAddr().String()}
	emsadminOpts.Logger = lgr
	if withAuth {
		emsadminOpts.AdminUsers = []string{"matt"}
	}
	emsadmin1, err := New(emsadminOpts)
	if err != nil {
		panic(err)
	}
	go func() {
		err := emsadmin1.Main()
		if err != nil {
			panic(err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	return tmpDir, []*emssvr.EMSD{emsd1}, []*emslookupd.EMSLookupd{emslookupd1}, emsadmin1
}

func TestPing(t *testing.T) {
	dataPath, emsds, emslookupds, emsadmin1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupds[0].Exit()
	defer emsadmin1.Exit()

	client := http.Client{}
	url := fmt.Sprintf("http://%s/ping", emsadmin1.RealHTTPAddr())
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	test.Equal(t, []byte("OK"), body)
}

func TestHTTPTopicsGET(t *testing.T) {
	dataPath, emsds, emslookupds, emsadmin1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupds[0].Exit()
	defer emsadmin1.Exit()

	topicName := "test_topics_get" + strconv.Itoa(int(time.Now().Unix()))
	emsds[0].GetTopic(topicName)
	time.Sleep(100 * time.Millisecond)

	client := http.Client{}
	url := fmt.Sprintf("http://%s/api/topics", emsadmin1.RealHTTPAddr())
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	tr := TopicsDoc{}
	err = json.Unmarshal(body, &tr)
	test.Nil(t, err)
	test.Equal(t, 1, len(tr.Topics))
	test.Equal(t, topicName, tr.Topics[0])
}

func TestHTTPTopicGET(t *testing.T) {
	dataPath, emsds, emslookupds, emsadmin1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupds[0].Exit()
	defer emsadmin1.Exit()

	topicName := "test_topic_get" + strconv.Itoa(int(time.Now().Unix()))
	emsds[0].GetTopic(topicName)
	time.Sleep(100 * time.Millisecond)

	client := http.Client{}
	url := fmt.Sprintf("http://%s/api/topics/%s", emsadmin1.RealHTTPAddr(), topicName)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	ts := TopicStatsDoc{}
	err = json.Unmarshal(body, &ts)
	test.Nil(t, err)
	test.Equal(t, topicName, ts.TopicName)
	test.Equal(t, 0, int(ts.Depth))
	test.Equal(t, 0, int(ts.MemoryDepth))
	test.Equal(t, 0, int(ts.BackendDepth))
	test.Equal(t, 0, int(ts.MessageCount))
	test.Equal(t, false, ts.Paused)
}

func TestHTTPNodesGET(t *testing.T) {
	dataPath, emsds, emslookupds, emsadmin1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupds[0].Exit()
	defer emsadmin1.Exit()

	time.Sleep(100 * time.Millisecond)

	client := http.Client{}
	url := fmt.Sprintf("http://%s/api/nodes", emsadmin1.RealHTTPAddr())
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	hostname, _ := os.Hostname()

	t.Logf("%s", body)
	ns := NodesDoc{}
	err = json.Unmarshal(body, &ns)
	test.Nil(t, err)
	test.Equal(t, 1, len(ns.Nodes))
	testNode := ns.Nodes[0]
	test.Equal(t, hostname, testNode.Hostname)
	test.Equal(t, "127.0.0.1", testNode.BroadcastAddress)
	test.Equal(t, emsds[0].RealTCPAddr().Port, testNode.TCPPort)
	test.Equal(t, emsds[0].RealHTTPAddr().Port, testNode.HTTPPort)
	test.Equal(t, version.Binary, testNode.Version)
	test.Equal(t, 0, len(testNode.Topics))
}

func TestHTTPChannelGET(t *testing.T) {
	dataPath, emsds, emslookupds, emsadmin1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupds[0].Exit()
	defer emsadmin1.Exit()

	topicName := "test_channel_get" + strconv.Itoa(int(time.Now().Unix()))
	topic := emsds[0].GetTopic(topicName)
	topic.GetChannel("ch")
	time.Sleep(100 * time.Millisecond)

	client := http.Client{}
	url := fmt.Sprintf("http://%s/api/topics/%s/ch", emsadmin1.RealHTTPAddr(), topicName)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	cs := ChannelStatsDoc{}
	err = json.Unmarshal(body, &cs)
	test.Nil(t, err)
	test.Equal(t, topicName, cs.TopicName)
	test.Equal(t, "ch", cs.ChannelName)
	test.Equal(t, 0, int(cs.Depth))
	test.Equal(t, 0, int(cs.MemoryDepth))
	test.Equal(t, 0, int(cs.BackendDepth))
	test.Equal(t, 0, int(cs.MessageCount))
	test.Equal(t, false, cs.Paused)
	test.Equal(t, 0, int(cs.InFlightCount))
	test.Equal(t, 0, int(cs.DeferredCount))
	test.Equal(t, 0, int(cs.RequeueCount))
	test.Equal(t, 0, int(cs.TimeoutCount))
	test.Equal(t, 0, len(cs.Clients))
}

func TestHTTPNodesSingleGET(t *testing.T) {
	dataPath, emsds, emslookupds, emsadmin1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupds[0].Exit()
	defer emsadmin1.Exit()

	topicName := "test_nodes_single_get" + strconv.Itoa(int(time.Now().Unix()))
	topic := emsds[0].GetTopic(topicName)
	topic.GetChannel("ch")
	time.Sleep(100 * time.Millisecond)

	client := http.Client{}
	url := fmt.Sprintf("http://%s/api/nodes/%s", emsadmin1.RealHTTPAddr(),
		emsds[0].RealHTTPAddr().String())
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("%s", body)
	ns := NodeStatsDoc{}
	err = json.Unmarshal(body, &ns)
	test.Nil(t, err)
	test.Equal(t, emsds[0].RealHTTPAddr().String(), ns.Node)
	test.Equal(t, 1, len(ns.TopicStats))
	testTopic := ns.TopicStats[0]
	test.Equal(t, topicName, testTopic.TopicName)
	test.Equal(t, 0, int(testTopic.Depth))
	test.Equal(t, 0, int(testTopic.MemoryDepth))
	test.Equal(t, 0, int(testTopic.BackendDepth))
	test.Equal(t, 0, int(testTopic.MessageCount))
	test.Equal(t, false, testTopic.Paused)
}

func TestHTTPCreateTopicPOST(t *testing.T) {
	dataPath, emsds, emslookupds, emsadmin1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupds[0].Exit()
	defer emsadmin1.Exit()

	time.Sleep(100 * time.Millisecond)

	topicName := "test_create_topic_post" + strconv.Itoa(int(time.Now().Unix()))

	client := http.Client{}
	url := fmt.Sprintf("http://%s/api/topics", emsadmin1.RealHTTPAddr())
	body, _ := json.Marshal(map[string]interface{}{
		"topic": topicName,
	})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	resp, err := client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
}

func TestHTTPCreateTopicChannelPOST(t *testing.T) {
	dataPath, emsds, emslookupds, emsadmin1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupds[0].Exit()
	defer emsadmin1.Exit()

	time.Sleep(100 * time.Millisecond)

	topicName := "test_create_topic_channel_post" + strconv.Itoa(int(time.Now().Unix()))

	client := http.Client{}
	url := fmt.Sprintf("http://%s/api/topics", emsadmin1.RealHTTPAddr())
	body, _ := json.Marshal(map[string]interface{}{
		"topic":   topicName,
		"channel": "ch",
	})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	resp, err := client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
}

func TestHTTPTombstoneTopicNodePOST(t *testing.T) {
	dataPath, emsds, emslookupds, emsadmin1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupds[0].Exit()
	defer emsadmin1.Exit()

	topicName := "test_tombstone_topic_node_post" + strconv.Itoa(int(time.Now().Unix()))
	emsds[0].GetTopic(topicName)
	time.Sleep(100 * time.Millisecond)

	client := http.Client{}
	url := fmt.Sprintf("http://%s/api/nodes/%s", emsadmin1.RealHTTPAddr(), emsds[0].RealHTTPAddr())
	body, _ := json.Marshal(map[string]interface{}{
		"topic": topicName,
	})
	req, _ := http.NewRequest("DELETE", url, bytes.NewBuffer(body))
	resp, err := client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
}

func TestHTTPDeleteTopicPOST(t *testing.T) {
	dataPath, emsds, emslookupds, emsadmin1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupds[0].Exit()
	defer emsadmin1.Exit()

	topicName := "test_delete_topic_post" + strconv.Itoa(int(time.Now().Unix()))
	emsds[0].GetTopic(topicName)
	time.Sleep(100 * time.Millisecond)

	client := http.Client{}
	url := fmt.Sprintf("http://%s/api/topics/%s", emsadmin1.RealHTTPAddr(), topicName)
	req, _ := http.NewRequest("DELETE", url, nil)
	resp, err := client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
}

func TestHTTPDeleteChannelPOST(t *testing.T) {
	dataPath, emsds, emslookupds, emsadmin1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupds[0].Exit()
	defer emsadmin1.Exit()

	topicName := "test_delete_channel_post" + strconv.Itoa(int(time.Now().Unix()))
	topic := emsds[0].GetTopic(topicName)
	topic.GetChannel("ch")
	time.Sleep(100 * time.Millisecond)

	client := http.Client{}
	url := fmt.Sprintf("http://%s/api/topics/%s/ch", emsadmin1.RealHTTPAddr(), topicName)
	req, _ := http.NewRequest("DELETE", url, nil)
	resp, err := client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
}

func TestHTTPPauseTopicPOST(t *testing.T) {
	dataPath, emsds, emslookupds, emsadmin1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupds[0].Exit()
	defer emsadmin1.Exit()

	topicName := "test_pause_topic_post" + strconv.Itoa(int(time.Now().Unix()))
	emsds[0].GetTopic(topicName)
	time.Sleep(100 * time.Millisecond)

	client := http.Client{}
	url := fmt.Sprintf("http://%s/api/topics/%s", emsadmin1.RealHTTPAddr(), topicName)
	body, _ := json.Marshal(map[string]interface{}{
		"action": "pause",
	})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	resp, err := client.Do(req)
	test.Nil(t, err)
	body, _ = ioutil.ReadAll(resp.Body)
	test.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	url = fmt.Sprintf("http://%s/api/topics/%s", emsadmin1.RealHTTPAddr(), topicName)
	body, _ = json.Marshal(map[string]interface{}{
		"action": "unpause",
	})
	req, _ = http.NewRequest("POST", url, bytes.NewBuffer(body))
	resp, err = client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
}

func TestHTTPPauseChannelPOST(t *testing.T) {
	dataPath, emsds, emslookupds, emsadmin1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupds[0].Exit()
	defer emsadmin1.Exit()

	topicName := "test_pause_channel_post" + strconv.Itoa(int(time.Now().Unix()))
	topic := emsds[0].GetTopic(topicName)
	topic.GetChannel("ch")
	time.Sleep(100 * time.Millisecond)

	client := http.Client{}
	url := fmt.Sprintf("http://%s/api/topics/%s/ch", emsadmin1.RealHTTPAddr(), topicName)
	body, _ := json.Marshal(map[string]interface{}{
		"action": "pause",
	})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	resp, err := client.Do(req)
	test.Nil(t, err)
	body, _ = ioutil.ReadAll(resp.Body)
	test.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	url = fmt.Sprintf("http://%s/api/topics/%s/ch", emsadmin1.RealHTTPAddr(), topicName)
	body, _ = json.Marshal(map[string]interface{}{
		"action": "unpause",
	})
	req, _ = http.NewRequest("POST", url, bytes.NewBuffer(body))
	resp, err = client.Do(req)
	test.Nil(t, err)
	test.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
}

func TestHTTPEmptyTopicPOST(t *testing.T) {
	dataPath, emsds, emslookupds, emsadmin1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupds[0].Exit()
	defer emsadmin1.Exit()

	topicName := "test_empty_topic_post" + strconv.Itoa(int(time.Now().Unix()))
	topic := emsds[0].GetTopic(topicName)
	topic.PutMessage(emssvr.NewMessage(emssvr.MessageID{}, []byte("1234")))
	test.Equal(t, int64(1), topic.Depth())
	time.Sleep(100 * time.Millisecond)

	client := http.Client{}
	url := fmt.Sprintf("http://%s/api/topics/%s", emsadmin1.RealHTTPAddr(), topicName)
	body, _ := json.Marshal(map[string]interface{}{
		"action": "empty",
	})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	resp, err := client.Do(req)
	test.Nil(t, err)
	body, _ = ioutil.ReadAll(resp.Body)
	test.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	test.Equal(t, int64(0), topic.Depth())
}

func TestHTTPEmptyChannelPOST(t *testing.T) {
	dataPath, emsds, emslookupds, emsadmin1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupds[0].Exit()
	defer emsadmin1.Exit()

	topicName := "test_empty_channel_post" + strconv.Itoa(int(time.Now().Unix()))
	topic := emsds[0].GetTopic(topicName)
	channel := topic.GetChannel("ch")
	channel.PutMessage(emssvr.NewMessage(emssvr.MessageID{}, []byte("1234")))

	time.Sleep(100 * time.Millisecond)
	test.Equal(t, int64(1), channel.Depth())

	client := http.Client{}
	url := fmt.Sprintf("http://%s/api/topics/%s/ch", emsadmin1.RealHTTPAddr(), topicName)
	body, _ := json.Marshal(map[string]interface{}{
		"action": "empty",
	})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	resp, err := client.Do(req)
	test.Nil(t, err)
	body, _ = ioutil.ReadAll(resp.Body)
	test.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	test.Equal(t, int64(0), channel.Depth())
}

func TestHTTPconfig(t *testing.T) {
	dataPath, emsds, emslookupds, emsadmin1 := bootstrapEMSCluster(t)
	defer os.RemoveAll(dataPath)
	defer emsds[0].Exit()
	defer emslookupds[0].Exit()
	defer emsadmin1.Exit()

	lopts := emslookupd.NewOptions()
	lopts.Logger = test.NewTestLogger(t)

	lopts1 := *lopts
	_, _, lookupd1 := mustStartEMSLookupd(&lopts1)
	defer lookupd1.Exit()
	lopts2 := *lopts
	_, _, lookupd2 := mustStartEMSLookupd(&lopts2)
	defer lookupd2.Exit()

	url := fmt.Sprintf("http://%s/config/emslookupd_http_addresses", emsadmin1.RealHTTPAddr())
	resp, err := http.Get(url)
	test.Nil(t, err)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	test.Equal(t, 200, resp.StatusCode)
	origaddrs := fmt.Sprintf(`["%s"]`, emslookupds[0].RealHTTPAddr().String())
	test.Equal(t, origaddrs, string(body))

	client := http.Client{}
	addrs := fmt.Sprintf(`["%s","%s"]`, lookupd1.RealHTTPAddr().String(), lookupd2.RealHTTPAddr().String())
	url = fmt.Sprintf("http://%s/config/emslookupd_http_addresses", emsadmin1.RealHTTPAddr())
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer([]byte(addrs)))
	test.Nil(t, err)
	resp, err = client.Do(req)
	test.Nil(t, err)
	defer resp.Body.Close()
	body, _ = ioutil.ReadAll(resp.Body)
	test.Equal(t, 200, resp.StatusCode)
	test.Equal(t, addrs, string(body))

	url = fmt.Sprintf("http://%s/config/log_level", emsadmin1.RealHTTPAddr())
	req, err = http.NewRequest("PUT", url, bytes.NewBuffer([]byte(`fatal`)))
	test.Nil(t, err)
	resp, err = client.Do(req)
	test.Nil(t, err)
	defer resp.Body.Close()
	body, _ = ioutil.ReadAll(resp.Body)
	test.Equal(t, 200, resp.StatusCode)
	test.Equal(t, LOG_FATAL, emsadmin1.getOpts().LogLevel)

	url = fmt.Sprintf("http://%s/config/log_level", emsadmin1.RealHTTPAddr())
	req, err = http.NewRequest("PUT", url, bytes.NewBuffer([]byte(`bad`)))
	test.Nil(t, err)
	resp, err = client.Do(req)
	test.Nil(t, err)
	defer resp.Body.Close()
	body, _ = ioutil.ReadAll(resp.Body)
	test.Equal(t, 400, resp.StatusCode)
}

func TestHTTPconfigCIDR(t *testing.T) {
	opts := NewOptions()
	opts.HTTPAddress = "127.0.0.1:0"
	opts.EMSLookupdHTTPAddresses = []string{"127.0.0.1:4161"}
	opts.Logger = test.NewTestLogger(t)
	opts.AllowConfigFromCIDR = "10.0.0.0/8"
	emsadmin, err := New(opts)
	test.Nil(t, err)
	go func() {
		err := emsadmin.Main()
		if err != nil {
			panic(err)
		}
	}()
	defer emsadmin.Exit()

	time.Sleep(100 * time.Millisecond)

	url := fmt.Sprintf("http://%s/config/emslookupd_http_addresses", emsadmin.RealHTTPAddr())
	resp, err := http.Get(url)
	test.Nil(t, err)
	defer resp.Body.Close()
	_, _ = ioutil.ReadAll(resp.Body)
	test.Equal(t, 403, resp.StatusCode)
}
