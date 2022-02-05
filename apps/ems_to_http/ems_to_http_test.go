package main

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

// This is a Bhojpur EMS client that reads the specified topic/channel
// and performs HTTP requests (GET/POST) to the specified endpoints

import (
	"reflect"
	"testing"
)

func TestParseCustomHeaders(t *testing.T) {
	type args struct {
		strs []string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		{
			"Valid Custom Headers",
			args{[]string{"header1: value1", "header2:value2", "header3:value3", "header4:value4"}},
			map[string]string{"header1": "value1", "header2": "value2", "header3": "value3", "header4": "value4"},
			false,
		},
		{
			"Invalid Custom Headers where key is present but no value",
			args{[]string{"header1:", "header2:value2", "header3: value3", "header4:value4"}},
			nil,
			true,
		},
		{
			"Invalid Custom Headers where key is not present but value is present",
			args{[]string{"header1: value1", ": value2", "header3:value3", "header4:value4"}},
			nil,
			true,
		},
		{
			"Invalid Custom Headers where key and value are not present but ':' is specified",
			args{[]string{"header1:value1", "header2:value2", ":", "header4:value4"}},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCustomHeaders(tt.args.strs)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCustomHeaders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseCustomHeaders() = %v, want %v", got, tt.want)
			}
		})
	}
}
