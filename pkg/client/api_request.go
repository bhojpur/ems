package client

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
	"net"
	"net/http"
	"time"
)

type deadlinedConn struct {
	Timeout time.Duration
	net.Conn
}

func (c *deadlinedConn) Read(b []byte) (n int, err error) {
	c.Conn.SetReadDeadline(time.Now().Add(c.Timeout))
	return c.Conn.Read(b)
}

func (c *deadlinedConn) Write(b []byte) (n int, err error) {
	c.Conn.SetWriteDeadline(time.Now().Add(c.Timeout))
	return c.Conn.Write(b)
}

type wrappedResp struct {
	Status     string      `json:"status_txt"`
	StatusCode int         `json:"status_code"`
	Data       interface{} `json:"data"`
}

// stores the result in the value pointed to by ret(must be a pointer)
func apiRequestNegotiateV1(httpclient *http.Client, method string, endpoint string, headers http.Header, ret interface{}) error {
	req, err := http.NewRequest(method, endpoint, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header[k] = v
	}

	req.Header.Add("Accept", "application/vnd.ems; version=1.0")

	resp, err := httpclient.Do(req)
	if err != nil {
		return err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("got response %s %q", resp.Status, respBody)
	}

	if len(respBody) == 0 {
		respBody = []byte("{}")
	}

	if resp.Header.Get("X-EMS-Content-Type") == "Bhojpur EMS; version=1.0" {
		return json.Unmarshal(respBody, ret)
	}

	wResp := &wrappedResp{
		Data: ret,
	}

	if err = json.Unmarshal(respBody, wResp); err != nil {
		return err
	}

	// wResp.StatusCode here is equal to resp.StatusCode, so ignore it
	return nil
}
