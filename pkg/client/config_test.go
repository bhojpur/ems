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
	"math/rand"
	"net"
	"reflect"
	"testing"
	"time"
)

func TestConfigSet(t *testing.T) {
	c := NewConfig()
	if err := c.Set("not a real config value", struct{}{}); err == nil {
		t.Error("No error when setting an invalid value")
	}
	if err := c.Set("tls_v1", "lol"); err == nil {
		t.Error("No error when setting `tls_v1` to an invalid value")
	}
	if err := c.Set("tls_v1", true); err != nil {
		t.Errorf("Error setting `tls_v1` config. %s", err)
	}

	if err := c.Set("tls-insecure-skip-verify", true); err != nil {
		t.Errorf("Error setting `tls-insecure-skip-verify` config. %v", err)
	}
	if c.TlsConfig.InsecureSkipVerify != true {
		t.Errorf("Error setting `tls-insecure-skip-verify` config: %v", c.TlsConfig)
	}
	if err := c.Set("tls-min-version", "tls1.2"); err != nil {
		t.Errorf("Error setting `tls-min-version` config: %s", err)
	}
	if err := c.Set("tls-min-version", "tls1.3"); err == nil {
		t.Error("No error when setting `tls-min-version` to an invalid value")
	}
	if err := c.Set("local_addr", &net.TCPAddr{}); err != nil {
		t.Errorf("Error setting `local_addr` config: %s", err)
	}
	if err := c.Set("local_addr", "1.2.3.4:27015"); err != nil {
		t.Errorf("Error setting `local_addr` config: %s", err)
	}
	if err := c.Set("dial_timeout", "5s"); err != nil {
		t.Errorf("Error setting `dial_timeout` config: %s", err)
	}
	if c.LocalAddr.String() != "1.2.3.4:27015" {
		t.Error("Failed to assign `local_addr` config")
	}
	if reflect.ValueOf(c.BackoffStrategy).Type().String() != "*ems.ExponentialStrategy" {
		t.Error("Failed to set default `exponential` backoff strategy")
	}
	if err := c.Set("backoff_strategy", "full_jitter"); err != nil {
		t.Errorf("Failed to assign `backoff_strategy` config: %v", err)
	}
	if reflect.ValueOf(c.BackoffStrategy).Type().String() != "*ems.FullJitterStrategy" {
		t.Error("Failed to set `full_jitter` backoff strategy")
	}
}

func TestConfigValidate(t *testing.T) {
	c := NewConfig()
	if err := c.Validate(); err != nil {
		t.Error("initialized config is invalid")
	}
	c.DeflateLevel = 100
	if err := c.Validate(); err == nil {
		t.Error("no error set for invalid value")
	}
}

func TestExponentialBackoff(t *testing.T) {
	expected := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		8 * time.Second,
		32 * time.Second,
	}
	backoffTest(t, expected, func(c *Config) BackoffStrategy {
		return &ExponentialStrategy{cfg: c}
	})
}

func TestFullJitterBackoff(t *testing.T) {
	expected := []time.Duration{
		724039541 * time.Nanosecond,
		1603903257 * time.Nanosecond,
		5232470547 * time.Nanosecond,
		21467499218 * time.Nanosecond,
	}
	backoffTest(t, expected, func(c *Config) BackoffStrategy {
		return &FullJitterStrategy{cfg: c, rng: rand.New(rand.NewSource(99))}
	})
}

func backoffTest(t *testing.T, expected []time.Duration, cb func(c *Config) BackoffStrategy) {
	config := NewConfig()
	attempts := []int{0, 1, 3, 5}
	s := cb(config)
	for i := range attempts {
		result := s.Calculate(attempts[i])
		if result != expected[i] {
			t.Fatalf("wrong backoff duration %v for attempt %d (should be %v)",
				result, attempts[i], expected[i])
		}
	}
}
