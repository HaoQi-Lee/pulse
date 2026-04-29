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

	// channel capacity is 1, first write fills it
	n, err := w.Write([]byte("msg1"))
	if n != 4 || err != nil {
		t.Fatalf("first write: n=%d err=%v", n, err)
	}

	// second write should be dropped but NOT block
	done := make(chan struct{})
	go func() {
		w.Write([]byte("msg2"))
		close(done)
	}()

	select {
	case <-done:
		// good
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
			// Close returned before we collected all messages
		case <-timeout:
			t.Fatalf("expected 3 messages, got %d", len(msgs))
		}
	}

	// Close should block until all messages are drained
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

	// fill the buffer
	w.Write([]byte("msg1"))

	// capture stderr
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

	// create a dummy closed connection
	dummy, _ := net.Dial("tcp", ln.Addr().String())
	<-accepted // consume the first accept
	dummy.Close()

	// reconnect should dial to w.Cs and return a working connection
	newConn := w.reconnect(dummy)
	defer newConn.Close()

	// verify the new connection is usable
	n, err := newConn.Write([]byte("test"))
	if err != nil || n != 4 {
		t.Fatalf("write on reconnected conn: n=%d err=%v", n, err)
	}
}

func TestFieldsValues(t *testing.T) {
	// set global state that fields() depends on
	m = Meta{Beat: "logback"}
	f = Fields{Project: "test-project", Service: "golang"}
	local, _ = time.LoadLocation("")
	hostname, _ = os.Hostname()
	pid = "42"

	flds := fields("hello world")

	if flds["message"] != "hello world" {
		t.Errorf("message = %v, want %q", flds["message"], "hello world")
	}
	if flds["host"] != hostname {
		t.Errorf("host = %v, want %q", flds["host"], hostname)
	}
	if flds["thread_name"] != "42" {
		t.Errorf("thread_name = %v, want %q", flds["thread_name"], "42")
	}
	if flds["logger_name"] == "" {
		t.Error("logger_name should not be empty")
	}

	meta, ok := flds["@metadata"].(Meta)
	if !ok || meta.Beat != "logback" {
		t.Errorf("@metadata = %v, want Meta{Beat:%q}", flds["@metadata"], "logback")
	}
	fieldsVal, ok := flds["fields"].(Fields)
	if !ok || fieldsVal.Project != "test-project" || fieldsVal.Service != "golang" {
		t.Errorf("fields = %v, want Fields{Project:%q, Service:%q}", flds["fields"], "test-project", "golang")
	}

	ts, ok := flds["@timestamp"].(string)
	if !ok || len(ts) == 0 {
		t.Error("@timestamp should be a non-empty string")
	}
}

func TestGetCaller(t *testing.T) {
	// getCaller(0) should return the file containing the call site
	result := getCaller(0)
	if result == "" {
		t.Fatal("getCaller returned empty string")
	}

	// should contain exactly one slash (last two path segments)
	count := strings.Count(result, "/")
	if count != 1 {
		t.Errorf("getCaller = %q, expected exactly 1 slash, got %d", result, count)
	}

	if !strings.HasSuffix(result, "logger.go") {
		t.Errorf("getCaller = %q, expected to end with logger.go", result)
	}
}
