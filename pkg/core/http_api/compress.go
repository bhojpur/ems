package http_api

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
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type compressResponseWriter struct {
	io.Writer
	http.ResponseWriter
	http.Hijacker
}

func (w *compressResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *compressResponseWriter) WriteHeader(c int) {
	w.ResponseWriter.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(c)
}

func (w *compressResponseWriter) Write(b []byte) (int, error) {
	h := w.ResponseWriter.Header()
	if h.Get("Content-Type") == "" {
		h.Set("Content-Type", http.DetectContentType(b))
	}
	h.Del("Content-Length")
	return w.Writer.Write(b)
}

// CompressHandler gzip compresses HTTP responses for clients that support it
// via the 'Accept-Encoding' header.
func CompressHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	L:
		for _, enc := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
			switch strings.TrimSpace(enc) {
			case "gzip":
				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Add("Vary", "Accept-Encoding")

				gw := gzip.NewWriter(w)
				defer gw.Close()

				h, hok := w.(http.Hijacker)
				if !hok { /* w is not Hijacker... oh well... */
					h = nil
				}

				w = &compressResponseWriter{
					Writer:         gw,
					ResponseWriter: w,
					Hijacker:       h,
				}

				break L
			case "deflate":
				w.Header().Set("Content-Encoding", "deflate")
				w.Header().Add("Vary", "Accept-Encoding")

				fw, _ := flate.NewWriter(w, flate.DefaultCompression)
				defer fw.Close()

				h, hok := w.(http.Hijacker)
				if !hok { /* w is not Hijacker... oh well... */
					h = nil
				}

				w = &compressResponseWriter{
					Writer:         fw,
					ResponseWriter: w,
					Hijacker:       h,
				}

				break L
			}
		}

		h.ServeHTTP(w, r)
	})
}
