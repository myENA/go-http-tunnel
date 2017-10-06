// Copyright (C) 2017 Michał Matczuk
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tunnel

import (
	"io"
	"net/http"

	"github.com/myENA/go-http-tunnel/log"
)

type closeWriter interface {
	CloseWrite() error
}

type closeReader interface {
	CloseRead() error
}

func transfer(dst io.Writer, src io.ReadCloser, logger log.Logger) {
	n, err := io.Copy(dst, src)
	if err != nil {
		logger.Log(
			"level", 2,
			"msg", "copy error",
			"err", err,
		)
	}

	if d, ok := dst.(closeWriter); ok {
		d.CloseWrite()
	}

	if s, ok := src.(closeReader); ok {
		s.CloseRead()
	} else {
		src.Close()
	}

	logger.Log(
		"level", 3,
		"action", "transferred",
		"bytes", n,
	)
}

func copyHeader(dst, src http.Header) {
	for k, v := range src {
		vv := make([]string, len(v))
		copy(vv, v)
		dst[k] = vv
	}
}

type countWriter struct {
	w     io.Writer
	count int64
}

func (cw *countWriter) Write(p []byte) (n int, err error) {
	n, err = cw.w.Write(p)
	cw.count += int64(n)
	return
}

type flushWriter struct {
	w io.Writer
}

func (fw flushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.w.Write(p)
	if f, ok := fw.w.(http.Flusher); ok {
		f.Flush()
	}
	return
}
