package pulse

import (
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"time"
)

// TcpWriter implements io.Writer with async TCP transport, buffering, and automatic reconnection.
type TcpWriter struct {
	Cs      string
	ch      chan []byte
	done    chan struct{}
	started atomic.Bool
}

// NewTcpWriter creates and returns an async TCP writer.
func NewTcpWriter(addr string, bufferSize int) *TcpWriter {
	if bufferSize <= 0 {
		bufferSize = 1024
	}
	return &TcpWriter{
		Cs:   addr,
		ch:   make(chan []byte, bufferSize),
		done: make(chan struct{}),
	}
}

// Write implements io.Writer. Data is enqueued to an async buffer and sent via TCP.
func (w *TcpWriter) Write(p []byte) (n int, err error) {
	w.startConsume()

	buf := make([]byte, len(p))
	copy(buf, p)

	select {
	case w.ch <- buf:
	default:
		fmt.Fprintln(os.Stderr, "[pulse] log dropped: buffer full")
	}
	return len(p), nil
}

func (w *TcpWriter) startConsume() {
	if w.started.CompareAndSwap(false, true) {
		go w.consume()
	}
}

func (w *TcpWriter) writeWithTimeout(conn net.Conn, data []byte, timeout time.Duration) error {
	conn.SetWriteDeadline(time.Now().Add(timeout))
	_, err := conn.Write(data)
	conn.SetWriteDeadline(time.Time{})
	return err
}

func (w *TcpWriter) reconnect(oldConn net.Conn) net.Conn {
	oldConn.Close()
	for {
		conn, err := net.Dial("tcp", w.Cs)
		if err != nil {
			fmt.Fprintln(os.Stderr, "[pulse] reconnect failed:", err)
			time.Sleep(2 * time.Second)
			continue
		}
		return conn
	}
}

// Close flushes the buffer and closes the TCP connection.
// If no writes were made, Close returns immediately.
func (w *TcpWriter) Close() {
	if !w.started.Load() {
		close(w.done)
		return
	}
	close(w.ch)
	<-w.done
}

func (w *TcpWriter) consume() {
	var conn net.Conn
	for {
		var err error
		conn, err = net.Dial("tcp", w.Cs)
		if err == nil {
			break
		}
		fmt.Fprintln(os.Stderr, "[pulse] initial connect failed:", err)
		time.Sleep(2 * time.Second)
	}

	keepAliveTicker := time.NewTicker(30 * time.Second)
	defer keepAliveTicker.Stop()

	for {
		select {
		case data, ok := <-w.ch:
			if !ok {
				conn.Close()
				close(w.done)
				return
			}
			if err := w.writeWithTimeout(conn, data, 10*time.Second); err != nil {
				fmt.Fprintln(os.Stderr, string(data))
				conn = w.reconnect(conn)
				w.writeWithTimeout(conn, data, 10*time.Second)
			}

		case <-keepAliveTicker.C:
			if err := w.writeWithTimeout(conn, []byte("\n"), 10*time.Second); err != nil {
				conn = w.reconnect(conn)
			}
		}
	}
}
