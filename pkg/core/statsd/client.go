package statsd

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
	"io"
)

type Client struct {
	w      io.Writer
	prefix string
}

func NewClient(w io.Writer, prefix string) *Client {
	return &Client{
		w:      w,
		prefix: prefix,
	}
}

func (c *Client) Incr(stat string, count int64) error {
	return c.send(stat, "%d|c", count)
}

func (c *Client) Decr(stat string, count int64) error {
	return c.send(stat, "%d|c", -count)
}

func (c *Client) Timing(stat string, delta int64) error {
	return c.send(stat, "%d|ms", delta)
}

func (c *Client) Gauge(stat string, value int64) error {
	return c.send(stat, "%d|g", value)
}

func (c *Client) send(stat string, format string, value int64) error {
	format = fmt.Sprintf("%s%s:%s\n", c.prefix, stat, format)
	_, err := fmt.Fprintf(c.w, format, value)
	return err
}
