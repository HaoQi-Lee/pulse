package pulseslog

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/HaoQi-Lee/pulse"
	"log/slog"
)

func slogStartTCPServer(t *testing.T) (string, chan []byte) {
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

func slogReadEntry(t *testing.T, received chan []byte) map[string]any {
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

func TestSlogHandlerEnabled(t *testing.T) {
	handler := NewHandler(pulse.Options{
		Project:  "test",
		Logstash: "127.0.0.1:0",
		Level:    "info",
	})
	defer handler.Close()

	ctx := context.Background()

	if handler.Enabled(ctx, slog.LevelDebug) {
		t.Error("debug should not be enabled when level is info")
	}
	if !handler.Enabled(ctx, slog.LevelInfo) {
		t.Error("info should be enabled")
	}
	if !handler.Enabled(ctx, slog.LevelWarn) {
		t.Error("warn should be enabled")
	}
	if !handler.Enabled(ctx, slog.LevelError) {
		t.Error("error should be enabled")
	}
}

func TestSlogHandlerHandle(t *testing.T) {
	addr, received := slogStartTCPServer(t)

	handler := NewHandler(pulse.Options{
		Project:  "demo",
		Logstash: addr,
	})
	defer handler.Close()

	logger := slog.New(handler)
	logger.Info("test message")

	result := slogReadEntry(t, received)

	if result["message"] != "test message" {
		t.Errorf("message = %v, want 'test message'", result["message"])
	}
	if result["level"] != "info" {
		t.Errorf("level = %v, want 'info'", result["level"])
	}
	if result["logger_name"] == "" {
		t.Error("logger_name should not be empty")
	}
}

func TestSlogHandlerWithAttrs(t *testing.T) {
	addr, received := slogStartTCPServer(t)

	handler := NewHandler(pulse.Options{
		Project:  "demo",
		Logstash: addr,
	})
	defer handler.Close()

	logger := slog.New(handler).With("request_id", "abc-123")
	logger.Warn("warning message")

	result := slogReadEntry(t, received)

	if result["message"] != "warning message" {
		t.Errorf("message = %v, want 'warning message'", result["message"])
	}
	if result["level"] != "warn" {
		t.Errorf("level = %v, want 'warn'", result["level"])
	}
	extra, ok := result["extra"].(map[string]any)
	if !ok {
		t.Fatal("extra is not a map")
	}
	if extra["request_id"] != "abc-123" {
		t.Errorf("extra.request_id = %v, want 'abc-123'", extra["request_id"])
	}
}

func TestSlogHandlerError(t *testing.T) {
	addr, received := slogStartTCPServer(t)

	handler := NewHandler(pulse.Options{
		Project:  "demo",
		Logstash: addr,
	})
	defer handler.Close()

	logger := slog.New(handler)
	logger.Error("something failed", "err", "timeout")

	result := slogReadEntry(t, received)

	if result["level"] != "error" {
		t.Errorf("level = %v, want 'error'", result["level"])
	}
	extra, _ := result["extra"].(map[string]any)
	if extra["err"] != "timeout" {
		t.Errorf("extra.err = %v, want 'timeout'", extra["err"])
	}
}

func TestSlogHandlerWithGroup(t *testing.T) {
	handler := NewHandler(pulse.Options{
		Project:  "test",
		Logstash: "127.0.0.1:0",
	})

	grouped := handler.WithGroup("module")
	if grouped == nil {
		t.Error("WithGroup should return a non-nil handler")
	}
}

func TestSlogHandlerClose(t *testing.T) {
	addr, received := slogStartTCPServer(t)

	handler := NewHandler(pulse.Options{
		Project:  "demo",
		Logstash: addr,
	})

	logger := slog.New(handler)
	logger.Info("before close")

	slogReadEntry(t, received)

	handler.Close()
}
