package pulsezerolog

import (
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/leehoawki/pulse"
	"github.com/rs/zerolog"
)

func zeroStartTCPServer(t *testing.T) (string, chan []byte) {
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

func zeroReadEntry(t *testing.T, received chan []byte) map[string]any {
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

func TestZerologHookRun(t *testing.T) {
	addr, received := zeroStartTCPServer(t)

	hook := NewHook(pulse.Options{
		Project:  "demo",
		Logstash: addr,
	})
	defer hook.Close()

	logger := zerolog.New(zerolog.Nop()).Hook(hook)
	logger.Info().Msg("test message")

	result := zeroReadEntry(t, received)

	if result["message"] != "test message" {
		t.Errorf("message = %v, want 'test message'", result["message"])
	}
	if result["level"] != "info" {
		t.Errorf("level = %v, want 'info'", result["level"])
	}
}

func TestZerologHookWarn(t *testing.T) {
	addr, received := zeroStartTCPServer(t)

	hook := NewHook(pulse.Options{
		Project:  "demo",
		Logstash: addr,
	})
	defer hook.Close()

	logger := zerolog.New(zerolog.Nop()).Hook(hook)
	logger.Warn().Msg("warning message")

	result := zeroReadEntry(t, received)

	if result["level"] != "warn" {
		t.Errorf("level = %v, want 'warn'", result["level"])
	}
	if result["message"] != "warning message" {
		t.Errorf("message = %v, want 'warning message'", result["message"])
	}
}

func TestZerologHookError(t *testing.T) {
	addr, received := zeroStartTCPServer(t)

	hook := NewHook(pulse.Options{
		Project:  "demo",
		Logstash: addr,
	})
	defer hook.Close()

	logger := zerolog.New(zerolog.Nop()).Hook(hook)
	logger.Error().Msg("something failed")

	result := zeroReadEntry(t, received)

	if result["level"] != "error" {
		t.Errorf("level = %v, want 'error'", result["level"])
	}
}

func TestZerologHookDebug(t *testing.T) {
	addr, received := zeroStartTCPServer(t)

	hook := NewHook(pulse.Options{
		Project:  "demo",
		Logstash: addr,
		Level:    "debug",
	})
	defer hook.Close()

	logger := zerolog.New(zerolog.Nop()).Level(zerolog.DebugLevel).Hook(hook)
	logger.Debug().Msg("debug message")

	result := zeroReadEntry(t, received)

	if result["level"] != "debug" {
		t.Errorf("level = %v, want 'debug'", result["level"])
	}
}

func TestZerologHookClose(t *testing.T) {
	addr, received := zeroStartTCPServer(t)

	hook := NewHook(pulse.Options{
		Project:  "demo",
		Logstash: addr,
	})

	logger := zerolog.New(zerolog.Nop()).Hook(hook)
	logger.Info().Msg("before close")

	zeroReadEntry(t, received)

	hook.Close()
}
