package lg

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
	"testing"

	"github.com/bhojpur/ems/pkg/core/test"
)

type mockLogger struct {
	Count int
}

func (l *mockLogger) Output(maxdepth int, s string) error {
	l.Count++
	return nil
}

func TestLogging(t *testing.T) {
	logger := &mockLogger{}

	// Test only fatal get through
	logger.Count = 0
	for i := 1; i <= 5; i++ {
		Logf(logger, FATAL, LogLevel(i), "Test")
	}
	test.Equal(t, 1, logger.Count)

	// Test only warnings or higher get through
	logger.Count = 0
	for i := 1; i <= 5; i++ {
		Logf(logger, WARN, LogLevel(i), "Test")
	}
	test.Equal(t, 3, logger.Count)

	// Test everything gets through
	logger.Count = 0
	for i := 1; i <= 5; i++ {
		Logf(logger, DEBUG, LogLevel(i), "Test")
	}
	test.Equal(t, 5, logger.Count)
}
