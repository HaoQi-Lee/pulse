package pulselogrus

import (
	"encoding/json"
	"net"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/leehoawki/pulse"
	"github.com/sirupsen/logrus"
)

func startTCPServer(t *testing.T) (string, chan []byte) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

	received := make(chan []byte, 20)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 8192)
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

	return ln.Addr().String(), received
}

func readLogEntry(t *testing.T, received chan []byte) map[string]any {
	t.Helper()
	select {
	case data := <-received:
		trimmed := strings.TrimSpace(string(data))
		var result map[string]any
		if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
			t.Fatalf("json unmarshal error: %v (raw: %q)", err, string(data))
		}
		return result
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for log entry")
		return nil
	}
}

func TestLogrusHookLevels(t *testing.T) {
	hook := NewHook(pulse.Options{
		Project:  "test",
		Logstash: "127.0.0.1:0",
		Level:    "info",
	})

	levels := hook.Levels()
	// info level should include: panic, fatal, error, warn, info
	expected := []logrus.Level{
		logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel,
		logrus.WarnLevel, logrus.InfoLevel,
	}
	if len(levels) != len(expected) {
		t.Fatalf("Levels() returned %d levels, want %d", len(levels), len(expected))
	}
	for i, l := range levels {
		if l != expected[i] {
			t.Errorf("Levels()[%d] = %v, want %v", i, l, expected[i])
		}
	}
}

func TestLogrusHookFire(t *testing.T) {
	addr, received := startTCPServer(t)

	hook := NewHook(pulse.Options{
		Project:  "demo",
		Logstash: addr,
	})
	defer hook.Close()

	entry := &logrus.Entry{
		Logger:  logrus.New(),
		Data:    logrus.Fields{"key": "value"},
		Time:    time.Now(),
		Level:   logrus.InfoLevel,
		Message: "test message",
	}

	if err := hook.Fire(entry); err != nil {
		t.Fatal(err)
	}

	result := readLogEntry(t, received)

	if result["message"] != "test message" {
		t.Errorf("message = %v, want 'test message'", result["message"])
	}
	if result["level"] != "info" {
		t.Errorf("level = %v, want 'info'", result["level"])
	}

	extra, ok := result["extra"].(map[string]any)
	if !ok {
		t.Fatal("extra is not a map")
	}
	if extra["key"] != "value" {
		t.Errorf("extra.key = %v, want 'value'", extra["key"])
	}
}

func TestLogrusHookFireWithCaller(t *testing.T) {
	addr, received := startTCPServer(t)

	hook := NewHook(pulse.Options{
		Project:  "demo",
		Logstash: addr,
	})
	defer hook.Close()

	caller := &runtime.Frame{
		File: "/home/user/project/cmd/server/main.go",
		Line: 42,
	}
	entry := &logrus.Entry{
		Logger:  logrus.New(),
		Data:    logrus.Fields{},
		Time:    time.Now(),
		Level:   logrus.InfoLevel,
		Message: "with caller",
		Caller:  caller,
	}

	if err := hook.Fire(entry); err != nil {
		t.Fatal(err)
	}

	result := readLogEntry(t, received)

	if result["logger_name"] != "server/main.go" {
		t.Errorf("logger_name = %v, want 'server/main.go'", result["logger_name"])
	}
}

func TestLogrusHookFireError(t *testing.T) {
	addr, received := startTCPServer(t)

	hook := NewHook(pulse.Options{
		Project:  "demo",
		Logstash: addr,
	})
	defer hook.Close()

	entry := &logrus.Entry{
		Logger:  logrus.New(),
		Data:    logrus.Fields{"err": "connection failed"},
		Time:    time.Now(),
		Level:   logrus.ErrorLevel,
		Message: "something went wrong",
	}

	if err := hook.Fire(entry); err != nil {
		t.Fatal(err)
	}

	result := readLogEntry(t, received)

	if result["level"] != "error" {
		t.Errorf("level = %v, want 'error'", result["level"])
	}
	if result["message"] != "something went wrong" {
		t.Errorf("message = %v, want 'something went wrong'", result["message"])
	}
}

func TestLogrusHookClose(t *testing.T) {
	addr, received := startTCPServer(t)

	hook := NewHook(pulse.Options{
		Project:  "demo",
		Logstash: addr,
	})

	entry := &logrus.Entry{
		Logger:  logrus.New(),
		Data:    logrus.Fields{},
		Time:    time.Now(),
		Level:   logrus.InfoLevel,
		Message: "before close",
	}
	hook.Fire(entry)

	// Read the entry to ensure it was written
	readLogEntry(t, received)

	hook.Close()
}
