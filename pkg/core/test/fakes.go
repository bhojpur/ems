package test

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
	"net"
	"time"
)

type FakeNetConn struct {
	ReadFunc             func([]byte) (int, error)
	WriteFunc            func([]byte) (int, error)
	CloseFunc            func() error
	LocalAddrFunc        func() net.Addr
	RemoteAddrFunc       func() net.Addr
	SetDeadlineFunc      func(time.Time) error
	SetReadDeadlineFunc  func(time.Time) error
	SetWriteDeadlineFunc func(time.Time) error
}

func (f FakeNetConn) Read(b []byte) (int, error)         { return f.ReadFunc(b) }
func (f FakeNetConn) Write(b []byte) (int, error)        { return f.WriteFunc(b) }
func (f FakeNetConn) Close() error                       { return f.CloseFunc() }
func (f FakeNetConn) LocalAddr() net.Addr                { return f.LocalAddrFunc() }
func (f FakeNetConn) RemoteAddr() net.Addr               { return f.RemoteAddrFunc() }
func (f FakeNetConn) SetDeadline(t time.Time) error      { return f.SetDeadlineFunc(t) }
func (f FakeNetConn) SetReadDeadline(t time.Time) error  { return f.SetReadDeadlineFunc(t) }
func (f FakeNetConn) SetWriteDeadline(t time.Time) error { return f.SetWriteDeadlineFunc(t) }

type fakeNetAddr struct{}

func (fakeNetAddr) Network() string { return "" }
func (fakeNetAddr) String() string  { return "" }

func NewFakeNetConn() FakeNetConn {
	netAddr := fakeNetAddr{}
	return FakeNetConn{
		ReadFunc:             func(b []byte) (int, error) { return 0, nil },
		WriteFunc:            func(b []byte) (int, error) { return len(b), nil },
		CloseFunc:            func() error { return nil },
		LocalAddrFunc:        func() net.Addr { return netAddr },
		RemoteAddrFunc:       func() net.Addr { return netAddr },
		SetDeadlineFunc:      func(time.Time) error { return nil },
		SetWriteDeadlineFunc: func(time.Time) error { return nil },
		SetReadDeadlineFunc:  func(time.Time) error { return nil },
	}
}
