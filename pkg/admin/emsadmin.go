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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"
	"sync/atomic"

	"github.com/bhojpur/ems/pkg/core/http_api"
	"github.com/bhojpur/ems/pkg/core/util"
	"github.com/bhojpur/ems/pkg/core/version"
)

type EMSAdmin struct {
	sync.RWMutex
	opts                atomic.Value
	httpListener        net.Listener
	waitGroup           util.WaitGroupWrapper
	notifications       chan *AdminAction
	graphiteURL         *url.URL
	httpClientTLSConfig *tls.Config
}

func New(opts *Options) (*EMSAdmin, error) {
	if opts.Logger == nil {
		opts.Logger = log.New(os.Stderr, opts.LogPrefix, log.Ldate|log.Ltime|log.Lmicroseconds)
	}

	n := &EMSAdmin{
		notifications: make(chan *AdminAction),
	}
	n.swapOpts(opts)

	if len(opts.EMSDHTTPAddresses) == 0 && len(opts.EMSLookupdHTTPAddresses) == 0 {
		return nil, errors.New("--emsd-http-address or --lookupd-http-address required")
	}

	if len(opts.EMSDHTTPAddresses) != 0 && len(opts.EMSLookupdHTTPAddresses) != 0 {
		return nil, errors.New("use --emsd-http-address or --lookupd-http-address not both")
	}

	if opts.HTTPClientTLSCert != "" && opts.HTTPClientTLSKey == "" {
		return nil, errors.New("--http-client-tls-key must be specified with --http-client-tls-cert")
	}

	if opts.HTTPClientTLSKey != "" && opts.HTTPClientTLSCert == "" {
		return nil, errors.New("--http-client-tls-cert must be specified with --http-client-tls-key")
	}

	n.httpClientTLSConfig = &tls.Config{
		InsecureSkipVerify: opts.HTTPClientTLSInsecureSkipVerify,
	}
	if opts.HTTPClientTLSCert != "" && opts.HTTPClientTLSKey != "" {
		cert, err := tls.LoadX509KeyPair(opts.HTTPClientTLSCert, opts.HTTPClientTLSKey)
		if err != nil {
			return nil, fmt.Errorf("failed to LoadX509KeyPair %s, %s - %s",
				opts.HTTPClientTLSCert, opts.HTTPClientTLSKey, err)
		}
		n.httpClientTLSConfig.Certificates = []tls.Certificate{cert}
	}
	if opts.HTTPClientTLSRootCAFile != "" {
		tlsCertPool := x509.NewCertPool()
		caCertFile, err := ioutil.ReadFile(opts.HTTPClientTLSRootCAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read TLS root CA file %s - %s",
				opts.HTTPClientTLSRootCAFile, err)
		}
		if !tlsCertPool.AppendCertsFromPEM(caCertFile) {
			return nil, fmt.Errorf("failed to AppendCertsFromPEM %s", opts.HTTPClientTLSRootCAFile)
		}
		n.httpClientTLSConfig.RootCAs = tlsCertPool
	}

	for _, address := range opts.EMSLookupdHTTPAddresses {
		_, err := net.ResolveTCPAddr("tcp", address)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve --lookupd-http-address (%s) - %s", address, err)
		}
	}

	for _, address := range opts.EMSDHTTPAddresses {
		_, err := net.ResolveTCPAddr("tcp", address)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve --emsd-http-address (%s) - %s", address, err)
		}
	}

	if opts.ProxyGraphite {
		url, err := url.Parse(opts.GraphiteURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse --graphite-url (%s) - %s", opts.GraphiteURL, err)
		}
		n.graphiteURL = url
	}

	if opts.AllowConfigFromCIDR != "" {
		_, _, err := net.ParseCIDR(opts.AllowConfigFromCIDR)
		if err != nil {
			return nil, fmt.Errorf("failed to parse --allow-config-from-cidr (%s) - %s", opts.AllowConfigFromCIDR, err)
		}
	}

	opts.BasePath = normalizeBasePath(opts.BasePath)

	n.logf(LOG_INFO, version.String("emsadmin"))

	var err error
	n.httpListener, err = net.Listen("tcp", n.getOpts().HTTPAddress)
	if err != nil {
		return nil, fmt.Errorf("listen (%s) failed - %s", n.getOpts().HTTPAddress, err)
	}

	return n, nil
}

func normalizeBasePath(p string) string {
	if len(p) == 0 {
		return "/"
	}
	// add leading slash
	if p[0] != '/' {
		p = "/" + p
	}
	return path.Clean(p)
}

func (n *EMSAdmin) getOpts() *Options {
	return n.opts.Load().(*Options)
}

func (n *EMSAdmin) swapOpts(opts *Options) {
	n.opts.Store(opts)
}

func (n *EMSAdmin) RealHTTPAddr() *net.TCPAddr {
	return n.httpListener.Addr().(*net.TCPAddr)
}

func (n *EMSAdmin) handleAdminActions() {
	for action := range n.notifications {
		content, err := json.Marshal(action)
		if err != nil {
			n.logf(LOG_ERROR, "failed to serialize admin action - %s", err)
		}
		httpclient := &http.Client{
			Transport: http_api.NewDeadlineTransport(n.getOpts().HTTPClientConnectTimeout, n.getOpts().HTTPClientRequestTimeout),
		}
		n.logf(LOG_INFO, "POSTing notification to %s", n.getOpts().NotificationHTTPEndpoint)
		resp, err := httpclient.Post(n.getOpts().NotificationHTTPEndpoint,
			"application/json", bytes.NewBuffer(content))
		if err != nil {
			n.logf(LOG_ERROR, "failed to POST notification - %s", err)
		}
		resp.Body.Close()
	}
}

func (n *EMSAdmin) Main() error {
	exitCh := make(chan error)
	var once sync.Once
	exitFunc := func(err error) {
		once.Do(func() {
			if err != nil {
				n.logf(LOG_FATAL, "%s", err)
			}
			exitCh <- err
		})
	}

	httpServer := NewHTTPServer(n)
	n.waitGroup.Wrap(func() {
		exitFunc(http_api.Serve(n.httpListener, http_api.CompressHandler(httpServer), "HTTP", n.logf))
	})
	n.waitGroup.Wrap(n.handleAdminActions)

	err := <-exitCh
	return err
}

func (n *EMSAdmin) Exit() {
	if n.httpListener != nil {
		n.httpListener.Close()
	}
	close(n.notifications)
	n.waitGroup.Wait()
}
