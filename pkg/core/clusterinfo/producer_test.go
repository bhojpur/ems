package clusterinfo

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

import "testing"

func TestHostNameAddresses(t *testing.T) {
	p := &Producer{
		BroadcastAddress: "host.domain.com",
		TCPPort:          4150,
		HTTPPort:         4151,
	}

	if p.HTTPAddress() != "host.domain.com:4151" {
		t.Errorf("Incorrect HTTPAddress: %s", p.HTTPAddress())
	}
	if p.TCPAddress() != "host.domain.com:4150" {
		t.Errorf("Incorrect TCPAddress: %s", p.TCPAddress())
	}
}

func TestIPv4Addresses(t *testing.T) {
	p := &Producer{
		BroadcastAddress: "192.168.1.17",
		TCPPort:          4150,
		HTTPPort:         4151,
	}

	if p.HTTPAddress() != "192.168.1.17:4151" {
		t.Errorf("Incorrect IPv4 HTTPAddress: %s", p.HTTPAddress())
	}
	if p.TCPAddress() != "192.168.1.17:4150" {
		t.Errorf("Incorrect IPv4 TCPAddress: %s", p.TCPAddress())
	}
}

func TestIPv6Addresses(t *testing.T) {
	p := &Producer{
		BroadcastAddress: "fd4a:622f:d2f2::1",
		TCPPort:          4150,
		HTTPPort:         4151,
	}
	if p.HTTPAddress() != "[fd4a:622f:d2f2::1]:4151" {
		t.Errorf("Incorrect IPv6 HTTPAddress: %s", p.HTTPAddress())
	}
	if p.TCPAddress() != "[fd4a:622f:d2f2::1]:4150" {
		t.Errorf("Incorrect IPv6 TCPAddress: %s", p.TCPAddress())
	}
}
