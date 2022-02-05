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
	"encoding/base64"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type AdminAction struct {
	Action    string `json:"action"`
	Topic     string `json:"topic"`
	Channel   string `json:"channel,omitempty"`
	Node      string `json:"node,omitempty"`
	Timestamp int64  `json:"timestamp"`
	User      string `json:"user,omitempty"`
	RemoteIP  string `json:"remote_ip"`
	UserAgent string `json:"user_agent"`
	URL       string `json:"url"` // The URL of the HTTP request that triggered this action
	Via       string `json:"via"` // the Hostname of the emsadmin performing this action
}

func basicAuthUser(req *http.Request) string {
	s := strings.SplitN(req.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 || s[0] != "Basic" {
		return ""
	}
	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return ""
	}
	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		return ""
	}
	return pair[0]
}

func (s *httpServer) notifyAdminAction(action, topic, channel, node string, req *http.Request) {
	if s.emsadmin.getOpts().NotificationHTTPEndpoint == "" {
		return
	}
	via, _ := os.Hostname()

	u := url.URL{
		Scheme:   "http",
		Host:     req.Host,
		Path:     req.URL.Path,
		RawQuery: req.URL.RawQuery,
	}
	if req.TLS != nil || req.Header.Get("X-Scheme") == "https" {
		u.Scheme = "https"
	}

	a := &AdminAction{
		Action:    action,
		Topic:     topic,
		Channel:   channel,
		Node:      node,
		Timestamp: time.Now().Unix(),
		User:      basicAuthUser(req),
		RemoteIP:  req.RemoteAddr,
		UserAgent: req.UserAgent(),
		URL:       u.String(),
		Via:       via,
	}
	// Perform all work in a new goroutine so this never blocks
	go func() { s.emsadmin.notifications <- a }()
}
