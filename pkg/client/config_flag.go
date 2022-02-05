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
	"strings"
)

// ConfigFlag wraps a Config and implements the flag.Value interface
type ConfigFlag struct {
	Config *Config
}

// Set takes a comma separated value and follows the rules in Config.Set
// using the first field as the option key, and the second (if present) as the value
func (c *ConfigFlag) Set(opt string) (err error) {
	parts := strings.SplitN(opt, ",", 2)
	key := parts[0]

	switch len(parts) {
	case 1:
		// default options specified without a value to boolean true
		err = c.Config.Set(key, true)
	case 2:
		err = c.Config.Set(key, parts[1])
	}
	return
}

// String implements the flag.Value interface
func (c *ConfigFlag) String() string {
	return ""
}
