package util

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

func BenchmarkUniqRands5of5(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UniqRands(5, 5)
	}
}
func BenchmarkUniqRands20of20(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UniqRands(20, 20)
	}
}

func BenchmarkUniqRands20of50(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UniqRands(20, 50)
	}
}

func TestUniqRands(t *testing.T) {
	var x []int
	x = UniqRands(3, 10)
	test.Equal(t, 3, len(x))

	x = UniqRands(10, 5)
	test.Equal(t, 5, len(x))

	x = UniqRands(10, 20)
	test.Equal(t, 10, len(x))
}
