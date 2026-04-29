package pulse

import (
	"bytes"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

func TestTcpWriterWriteDoesNotBlock(t *testing.T) {
	w := &TcpWriter{
		Cs:   "localhost:0",
		ch:   make(chan []byte, 1),
		done: make(chan struct{}),
	}

	n, err := w.Write([]byte("msg1"))
	if n != 4 || err != nil {
		t.Fatalf("first write: n=%d err=%v", n, err)
	}

	done := make(chan struct{})
	go func() {
		w.Write([]byte("msg2"))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Write blocked when buffer full")
	}
}

func TestTcpWriterConsumesAndWrites(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()
	received := make(chan []byte, 10)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				received <- append([]byte{}, buf[:n]...)
			}
			if err != nil {
				return
			}
		}
	}()

	w := &TcpWriter{
		Cs:   addr,
		ch:   make(chan []byte, 100),
		done: make(chan struct{}),
	}

	w.Write([]byte("hello"))
	w.Write([]byte("world"))

	timeout := time.After(3 * time.Second)
	var msgs [][]byte
	for len(msgs) < 2 {
		select {
		case m := <-received:
			msgs = append(msgs, m)
		case <-timeout:
			t.Fatalf("expected 2 messages, got %d", len(msgs))
		}
	}

	if string(msgs[0]) != "hello" {
		t.Errorf("first msg = %q, want %q", msgs[0], "hello")
	}
	if string(msgs[1]) != "world" {
		t.Errorf("second msg = %q, want %q", msgs[1], "world")
	}
}

func TestWriteWithTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				// read and discard
			}
			if err != nil {
				return
			}
		}
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	w := &TcpWriter{
		Cs: ln.Addr().String(),
		ch: make(chan []byte, 100),
	}

	err = w.writeWithTimeout(conn, []byte("hello"), 2*time.Second)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestKeepAliveSent(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	received := make(chan []byte, 10)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				received <- append([]byte{}, buf[:n]...)
			}
			if err != nil {
				return
			}
		}
	}()

	w := &TcpWriter{
		Cs: ln.Addr().String(),
		ch: make(chan []byte, 100),
	}

	w.Write([]byte("first"))

	select {
	case <-received:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for first message")
	}

	w.Write([]byte("second"))

	select {
	case <-received:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for second message")
	}
}

func TestCloseFlushesBuffer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	received := make(chan []byte, 10)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				received <- append([]byte{}, buf[:n]...)
			}
			if err != nil {
				return
			}
		}
	}()

	w := &TcpWriter{
		Cs:   ln.Addr().String(),
		ch:   make(chan []byte, 100),
		done: make(chan struct{}),
	}

	w.Write([]byte("msg1"))
	w.Write([]byte("msg2"))
	w.Write([]byte("msg3"))

	done := make(chan struct{})
	go func() {
		w.Close()
		close(done)
	}()

	timeout := time.After(5 * time.Second)
	var msgs [][]byte
	for len(msgs) < 3 {
		select {
		case m := <-received:
			msgs = append(msgs, m)
		case <-done:
		case <-timeout:
			t.Fatalf("expected 3 messages, got %d", len(msgs))
		}
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return after flushing buffer")
	}

	got := string(bytes.Join(msgs, []byte{}))
	for _, want := range []string{"msg1", "msg2", "msg3"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing message %q in output %q", want, got)
		}
	}
}

func TestWriteDroppedLogsToStderr(t *testing.T) {
	w := &TcpWriter{
		Cs:   "localhost:0",
		ch:   make(chan []byte, 1),
		done: make(chan struct{}),
	}

	w.Write([]byte("msg1"))

	old := os.Stderr
	r, wStderr, _ := os.Pipe()
	os.Stderr = wStderr

	w.Write([]byte("msg2"))

	wStderr.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "log dropped") {
		t.Errorf("expected stderr warning about dropped log, got %q", output)
	}
}

func TestReconnectDialsNewConnection(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		accepted <- conn
	}()

	w := &TcpWriter{
		Cs:   ln.Addr().String(),
		ch:   make(chan []byte, 100),
		done: make(chan struct{}),
	}

	dummy, _ := net.Dial("tcp", ln.Addr().String())
	<-accepted
	dummy.Close()

	newConn := w.reconnect(dummy)
	defer newConn.Close()

	n, err := newConn.Write([]byte("test"))
	if err != nil || n != 4 {
		t.Fatalf("write on reconnected conn: n=%d err=%v", n, err)
	}
}

func TestNewTcpWriter(t *testing.T) {
	w := NewTcpWriter("127.0.0.1:0", 0)
	if w == nil {
		t.Fatal("NewTcpWriter returned nil")
	}
	if w.Cs != "127.0.0.1:0" {
		t.Errorf("Cs = %q, want %q", w.Cs, "127.0.0.1:0")
	}
	if cap(w.ch) != 1024 {
		t.Errorf("buffer cap = %d, want %d", cap(w.ch), 1024)
	}
}

func TestNewTcpWriterCustomBufferSize(t *testing.T) {
	w := NewTcpWriter("127.0.0.1:0", 2048)
	if cap(w.ch) != 2048 {
		t.Errorf("buffer cap = %d, want %d", cap(w.ch), 2048)
	}
}

func TestCloseWithoutWrite(t *testing.T) {
	w := NewTcpWriter("localhost:0", 100)

	done := make(chan struct{})
	go func() {
		w.Close()
		close(done)
	}()

	select {
	case <-done:
		// Close returned immediately — good
	case <-time.After(time.Second):
		t.Fatal("Close hung when no writes were made")
	}
}
