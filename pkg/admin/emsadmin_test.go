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
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/bhojpur/ems/pkg/core/lg"
	"github.com/bhojpur/ems/pkg/core/test"
	emssvr "github.com/bhojpur/ems/pkg/engine"
)

func TestNeitherEMSDAndEMSLookup(t *testing.T) {
	opts := NewOptions()
	opts.Logger = lg.NilLogger{}
	opts.HTTPAddress = "127.0.0.1:0"
	_, err := New(opts)
	test.NotNil(t, err)
	test.Equal(t, "--emsd-http-address or --lookupd-http-address required", fmt.Sprintf("%s", err))
}

func TestBothEMSDAndEMSLookup(t *testing.T) {
	opts := NewOptions()
	opts.Logger = lg.NilLogger{}
	opts.HTTPAddress = "127.0.0.1:0"
	opts.EMSLookupdHTTPAddresses = []string{"127.0.0.1:4161"}
	opts.EMSDHTTPAddresses = []string{"127.0.0.1:4151"}
	_, err := New(opts)
	test.NotNil(t, err)
	test.Equal(t, "use --emsd-http-address or --lookupd-http-address not both", fmt.Sprintf("%s", err))
}

func TestTLSHTTPClient(t *testing.T) {
	lgr := test.NewTestLogger(t)

	emsdOpts := emssvr.NewOptions()
	emsdOpts.TLSCert = "./test/server.pem"
	emsdOpts.TLSKey = "./test/server.key"
	emsdOpts.TLSRootCAFile = "./test/ca.pem"
	emsdOpts.TLSClientAuthPolicy = "require-verify"
	emsdOpts.Logger = lgr
	_, emsdHTTPAddr, emsd := mustStartEMSD(emsdOpts)
	defer os.RemoveAll(emsdOpts.DataPath)
	defer emsd.Exit()

	opts := NewOptions()
	opts.HTTPAddress = "127.0.0.1:0"
	opts.EMSDHTTPAddresses = []string{emsdHTTPAddr.String()}
	opts.HTTPClientTLSRootCAFile = "./test/ca.pem"
	opts.HTTPClientTLSCert = "./test/client.pem"
	opts.HTTPClientTLSKey = "./test/client.key"
	opts.Logger = lgr
	emsadmin, err := New(opts)
	test.Nil(t, err)
	go func() {
		err := emsadmin.Main()
		if err != nil {
			panic(err)
		}
	}()
	defer emsadmin.Exit()

	httpAddr := emsadmin.RealHTTPAddr()
	u := url.URL{
		Scheme: "http",
		Host:   httpAddr.String(),
		Path:   "/api/nodes/" + emsdHTTPAddr.String(),
	}

	resp, err := http.Get(u.String())
	test.Nil(t, err)
	defer resp.Body.Close()

	test.Equal(t, resp.StatusCode < 500, true)
}

func mustStartEMSD(opts *emssvr.Options) (*net.TCPAddr, *net.TCPAddr, *emssvr.EMSD) {
	opts.TCPAddress = "127.0.0.1:0"
	opts.HTTPAddress = "127.0.0.1:0"
	opts.HTTPSAddress = "127.0.0.1:0"
	if opts.DataPath == "" {
		tmpDir, err := ioutil.TempDir("", "ems-test-")
		if err != nil {
			panic(err)
		}
		opts.DataPath = tmpDir
	}
	emsd, err := emssvr.New(opts)
	if err != nil {
		panic(err)
	}
	go func() {
		err := emsd.Main()
		if err != nil {
			panic(err)
		}
	}()
	return emsd.RealTCPAddr(), emsd.RealHTTPAddr(), emsd
}
