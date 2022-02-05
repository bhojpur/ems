package protocol

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

type ChildErr interface {
	Parent() error
}

// ClientErr provides a way for Bhojpur EMS daemons to log a human reabable
// error string and return a machine readable string to the client.
type ClientErr struct {
	ParentErr error
	Code      string
	Desc      string
}

// Error returns the machine readable form
func (e *ClientErr) Error() string {
	return e.Code + " " + e.Desc
}

// Parent returns the parent error
func (e *ClientErr) Parent() error {
	return e.ParentErr
}

// NewClientErr creates a ClientErr with the supplied human and machine readable strings
func NewClientErr(parent error, code string, description string) *ClientErr {
	return &ClientErr{parent, code, description}
}

type FatalClientErr struct {
	ParentErr error
	Code      string
	Desc      string
}

// Error returns the machine readable form
func (e *FatalClientErr) Error() string {
	return e.Code + " " + e.Desc
}

// Parent returns the parent error
func (e *FatalClientErr) Parent() error {
	return e.ParentErr
}

// NewClientErr creates a ClientErr with the supplied human and machine readable strings
func NewFatalClientErr(parent error, code string, description string) *FatalClientErr {
	return &FatalClientErr{parent, code, description}
}
