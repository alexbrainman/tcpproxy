// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The tcpproxy command displays all data sent between tcp client and server.
//
// Run without parameters for usage information.
package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

func doCopy(id int, prefix byte, to io.WriteCloser, from io.ReadCloser, disconnectedBy chan string, peerName string) {
	defer func() {
		from.Close()
		to.Close()
		disconnectedBy <- peerName
	}()
	b := make([]byte, 4*1024)
	for {
		n, err := from.Read(b)
		if err != nil || n == 0 {
			// TODO: maybe display this error message in the log
			return
		}
		b2 := b[:n]
		ow.writeData(id, prefix, b2)
		for len(b2) > 0 {
			sent, err := to.Write(b2)
			if err != nil {
				// TODO: maybe display this error message in the log
				return
			}
			b2 = b2[sent:]
		}
	}
}

func serve(from net.Conn, raddr string) {
	defer from.Close()

	f := strings.Split(from.RemoteAddr().String(), ":")
	if len(f) != 2 {
		panic(fmt.Sprintf("Invalid remote address of client connection (%s)", from.RemoteAddr()))
	}
	id, err := strconv.Atoi(f[1])
	if err != nil {
		panic(fmt.Sprintf("Remote address port of client connection must be integer (%s): %v", from.RemoteAddr(), err))
	}
	ow.writeComment(id, fmt.Sprintf("client connected (%s)", from.RemoteAddr()))

	to, err := net.Dial("tcp", raddr)
	if err != nil {
		ow.writeComment(id, fmt.Sprintf("failed to connect to server: %v", err))
		return
	}
	s := fmt.Sprintf("(%s->%s)", from.RemoteAddr(), to.LocalAddr())
	ow.writeComment(id, "server connected "+s)

	c := make(chan string, 2)
	go doCopy(id, '|', to, from, c, "client")
	go doCopy(id, '-', from, to, c, "server")
	ow.writeComment(id, "disconnected by "+<-c+" "+s)
}

func runListener(laddr, raddr string) error {
	l, err := net.Listen("tcp", laddr)
	if err != nil {
		return err
	}
	defer l.Close()

	for {
		s, err := l.Accept()
		if err != nil {
			return err
		}
		go serve(s, raddr)
	}
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s <laddr>:<lport> <toaddr>:<toport>\n\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprint(os.Stderr, "\n  use :<lport> to listen on any address.\n")
		os.Exit(2)
	}
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
	}

	err := runListener(flag.Arg(0), flag.Arg(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
