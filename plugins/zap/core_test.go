package pulsezap

import (
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/HaoQi-Lee/pulse"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func zapStartTCPServer(t *testing.T) (string, chan []byte) {
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

func zapReadEntry(t *testing.T, received chan []byte) map[string]any {
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

func TestZapCoreEnabled(t *testing.T) {
	core := NewCore(pulse.Options{
		Project:  "test",
		Logstash: "127.0.0.1:0",
		Level:    "info",
	})
	defer core.Close()

	if core.Enabled(zapcore.DebugLevel) {
		t.Error("debug should not be enabled when level is info")
	}
	if !core.Enabled(zapcore.InfoLevel) {
		t.Error("info should be enabled")
	}
	if !core.Enabled(zapcore.WarnLevel) {
		t.Error("warn should be enabled")
	}
	if !core.Enabled(zapcore.ErrorLevel) {
		t.Error("error should be enabled")
	}
}

func TestZapCoreWrite(t *testing.T) {
	addr, received := zapStartTCPServer(t)

	core := NewCore(pulse.Options{
		Project:  "demo",
		Logstash: addr,
	})
	defer core.Close()

	logger := zap.New(core)
	logger.Info("test message")

	result := zapReadEntry(t, received)

	if result["message"] != "test message" {
		t.Errorf("message = %v, want 'test message'", result["message"])
	}
	if result["level"] != "info" {
		t.Errorf("level = %v, want 'info'", result["level"])
	}
}

func TestZapCoreWithCaller(t *testing.T) {
	addr, received := zapStartTCPServer(t)

	core := NewCore(pulse.Options{
		Project:  "demo",
		Logstash: addr,
	})
	defer core.Close()

	logger := zap.New(core, zap.AddCaller())
	logger.Info("with caller")

	result := zapReadEntry(t, received)

	if result["logger_name"] == "" {
		t.Error("logger_name should not be empty when caller is enabled")
	}
}

func TestZapCoreWithFields(t *testing.T) {
	addr, received := zapStartTCPServer(t)

	core := NewCore(pulse.Options{
		Project:  "demo",
		Logstash: addr,
	})
	defer core.Close()

	logger := zap.New(core).With(zap.String("request_id", "abc-123"))
	logger.Warn("warning message", zap.Int("retry", 3))

	result := zapReadEntry(t, received)

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

func TestZapCoreError(t *testing.T) {
	addr, received := zapStartTCPServer(t)

	core := NewCore(pulse.Options{
		Project:  "demo",
		Logstash: addr,
	})
	defer core.Close()

	logger := zap.New(core)
	logger.Error("something failed", zap.String("err", "timeout"))

	result := zapReadEntry(t, received)

	if result["level"] != "error" {
		t.Errorf("level = %v, want 'error'", result["level"])
	}
}

func TestZapCoreSync(t *testing.T) {
	core := NewCore(pulse.Options{
		Project:  "test",
		Logstash: "127.0.0.1:0",
	})
	defer core.Close()

	if err := core.Sync(); err != nil {
		t.Errorf("Sync() returned error: %v", err)
	}
}

func TestZapCoreClose(t *testing.T) {
	addr, received := zapStartTCPServer(t)

	core := NewCore(pulse.Options{
		Project:  "demo",
		Logstash: addr,
	})

	logger := zap.New(core)
	logger.Info("before close")

	zapReadEntry(t, received)

	core.Close()
}

// Test observer integration
func TestZapCoreCheck(t *testing.T) {
	core := NewCore(pulse.Options{
		Project:  "test",
		Logstash: "127.0.0.1:0",
		Level:    "warn",
	})
	defer core.Close()

	// Create a test entry
	entry := zapcore.Entry{Level: zapcore.InfoLevel}
	ce := &zapcore.CheckedEntry{}

	result := core.Check(entry, ce)
	// Info level should not be added since level is warn
	if result != ce {
		t.Error("Check should return same CheckedEntry for disabled level")
	}

	entry = zapcore.Entry{Level: zapcore.ErrorLevel}
	result = core.Check(entry, ce)
	// Error level should be added
	if result == nil {
		t.Error("Check should return non-nil for enabled level")
	}
}

// Ensure unused imports are consumed
var _ zapcore.Core
