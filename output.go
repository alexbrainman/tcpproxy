// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	isText      = flag.Bool("text", false, "output text insted of hex")
	columnWidth = flag.Int("width", 18, "number of hex columns")
)

type hexLineWriter struct {
	dest   io.Writer
	prefix string
	buf    []byte
}

func newHexLineWriter(dest io.Writer, prefix string) *hexLineWriter {
	return &hexLineWriter{
		dest:   dest,
		prefix: prefix,
		buf:    make([]byte, 0, *columnWidth),
	}
}

func (lw *hexLineWriter) write(b []byte) {
	for _, ch := range b {
		if len(lw.buf) == *columnWidth {
			lw.flush()
		}
		if len(lw.buf) == 0 {
			fmt.Fprintf(lw.dest, "%s ", lw.prefix)
		}
		fmt.Fprintf(lw.dest, " %2.2x", ch)
		lw.buf = append(lw.buf, ch)
	}
}

func (lw *hexLineWriter) flush() {
	if lw == nil {
		return
	}
	if len(lw.buf) > 0 {
		fmt.Fprintf(lw.dest, "%s ", strings.Repeat("   ", cap(lw.buf)-len(lw.buf)))
		for _, ch := range lw.buf {
			if ch < 32 || 126 < ch {
				// anything not ascii will be displayed as "."
				ch = '.'
			}
			fmt.Fprintf(lw.dest, "%c", ch)
		}
		lw.buf = lw.buf[:0]
	}
	fmt.Fprintf(lw.dest, "\n")
}

type textLineWriter struct {
	dest                 io.Writer
	prefix               string
	isPrefixWritePending bool
}

func newTextLineWriter(dest io.Writer, prefix string) *textLineWriter {
	fmt.Fprintf(dest, "%s ", prefix)
	return &textLineWriter{
		dest:                 dest,
		prefix:               prefix,
		isPrefixWritePending: false,
	}
}

func (lw *textLineWriter) write(b []byte) {
	for _, ch := range b {
		if lw.isPrefixWritePending {
			fmt.Fprintf(lw.dest, "%s ", lw.prefix)
			lw.isPrefixWritePending = false
		}
		fmt.Fprintf(lw.dest, "%c", ch)
		if ch == '\n' {
			lw.isPrefixWritePending = true
		}
	}
}

func (lw *textLineWriter) flush() {
}

type commentMsg struct {
	id  int
	str string
}

type dataMsg struct {
	id     int
	prefix byte
	b      []byte
}

type outputWriter struct {
	comments chan commentMsg
	data     chan dataMsg
}

var ow = newOutputWriter(os.Stdout)

func (w *outputWriter) writeComment(id int, s string) {
	ow.comments <- commentMsg{id, s}
}

func (w *outputWriter) writeData(id int, prefix byte, b []byte) {
	ow.data <- dataMsg{id, prefix, b}
}

func makePrefix(id int, prefix byte) string {
	return fmt.Sprintf("%4.4d %c", id, prefix)
}

type lineWriter interface {
	write([]byte)
	flush()
}

func (w *outputWriter) output(dest io.Writer) {
	var curId int
	var curPrefix byte
	var lw lineWriter
	for {
		select {
		case msg := <-w.comments:
			if lw != nil {
				lw.flush()
				lw = nil
			}
			p := makePrefix(msg.id, '*')
			fmt.Fprintf(dest, "%s\n%s %s\n%s\n", p, p, msg.str, p)
		case msg := <-w.data:
			if lw != nil && curId == msg.id && curPrefix == msg.prefix {
				lw.write(msg.b)
				continue
			}
			if lw != nil {
				lw.flush()
			}
			curId = msg.id
			curPrefix = msg.prefix
			p := makePrefix(msg.id, msg.prefix)
			if *isText {
				lw = newTextLineWriter(dest, p)
			} else {
				lw = newHexLineWriter(dest, p)
			}
			lw.write(msg.b)
		}
	}
}

func newOutputWriter(dest io.Writer) *outputWriter {
	w := &outputWriter{
		comments: make(chan commentMsg),
		data:     make(chan dataMsg),
	}
	go w.output(dest)
	return w
}
