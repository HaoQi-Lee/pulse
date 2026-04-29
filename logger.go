package pulse

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

var m Meta
var f Fields
var local *time.Location
var hostname string
var pid string

func Setup(name string, logstash string) func() {
	m = Meta{Beat: "logback"}
	f = Fields{Project: name, Service: "golang"}
	local, _ = time.LoadLocation("")
	hostname, _ = os.Hostname()
	pid = strconv.Itoa(os.Getpid())

	w := &TcpWriter{
		Cs:   logstash,
		ch:   make(chan []byte, 1024),
		done: make(chan struct{}),
	}

	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(w)
	logrus.SetLevel(logrus.InfoLevel)

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		w.Close()
		os.Exit(0)
	}()

	return w.Close
}

type Fields struct {
	Project string `json:"project"`
	Service string `json:"service"`
}

type Meta struct {
	Beat string `json:"beat"`
}

type TcpWriter struct {
	Cs   string
	ch   chan []byte
	done chan struct{}
	once sync.Once
}

func (w *TcpWriter) Write(p []byte) (n int, err error) {
	w.once.Do(func() { go w.consume() })

	buf := make([]byte, len(p))
	copy(buf, p)

	select {
	case w.ch <- buf:
	default:
		fmt.Fprintln(os.Stderr, "[x-sonar] log dropped: buffer full")
	}
	return len(p), nil
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
			fmt.Fprintln(os.Stderr, "[x-sonar] reconnect failed:", err)
			time.Sleep(2 * time.Second)
			continue
		}
		return conn
	}
}

func (w *TcpWriter) Close() {
	w.once.Do(func() { go w.consume() })
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
		fmt.Fprintln(os.Stderr, "[x-sonar] initial connect failed:", err)
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

func Debug(message string) {
	logrus.WithFields(fields(message)).Debug()
}

func Info(message string) {
	logrus.WithFields(fields(message)).Info()
}

func Warn(message string) {
	logrus.WithFields(fields(message)).Warn()
}

func Error(err error) {
	logrus.WithFields(fields(err.Error())).Error()
}

func fields(message string) logrus.Fields {
	return logrus.Fields{
		"thread_name": pid,
		"host":        hostname,
		"@timestamp":  time.Now().In(local).Format("2006-01-02T15:04:05.999Z"),
		"logger_name": getCaller(3),
		"@metadata":   m,
		"fields":      f,
		"message":     message,
	}
}

func getCaller(skip int) string {
	_, file, _, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	n := 0
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			n++
			if n >= 2 {
				file = file[i+1:]
				break
			}
		}
	}
	return file
}
