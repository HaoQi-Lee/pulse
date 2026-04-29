package pulse

import (
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"
)

func TestNewCore(t *testing.T) {
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
			}
			if err != nil {
				return
			}
		}
	}()

	c := New(Options{
		Project:  "test",
		Logstash: ln.Addr().String(),
	})
	defer c.Close()

	if c.Writer == nil {
		t.Error("Writer should not be nil")
	}
	if c.hostname == "" {
		t.Error("hostname should not be empty")
	}
	if c.pid == "" {
		t.Error("pid should not be empty")
	}
}

func TestBuildEntry(t *testing.T) {
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
			}
			if err != nil {
				return
			}
		}
	}()

	c := New(Options{
		Project:  "test-project",
		Logstash: ln.Addr().String(),
	})
	defer c.Close()

	entry := c.BuildEntry(0)

	if entry.ThreadName != c.pid {
		t.Errorf("ThreadName = %q, want %q", entry.ThreadName, c.pid)
	}
	if entry.Host != c.hostname {
		t.Errorf("Host = %q, want %q", entry.Host, c.hostname)
	}
	if entry.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
	if entry.LoggerName == "" {
		t.Error("LoggerName should not be empty")
	}
	if entry.Metadata.Beat != "logback" {
		t.Errorf("Metadata.Beat = %q, want %q", entry.Metadata.Beat, "logback")
	}
	if entry.Fields.Project != "test-project" {
		t.Errorf("Fields.Project = %q, want %q", entry.Fields.Project, "test-project")
	}
	if entry.Fields.Service != "golang" {
		t.Errorf("Fields.Service = %q, want %q", entry.Fields.Service, "golang")
	}
}

func TestBuildEntryCustomOptions(t *testing.T) {
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
			}
			if err != nil {
				return
			}
		}
	}()

	c := New(Options{
		Project:  "my-project",
		Service:  "my-service",
		Beat:     "custom-beat",
		Logstash: ln.Addr().String(),
	})
	defer c.Close()

	entry := c.BuildEntry(0)

	if entry.Fields.Project != "my-project" {
		t.Errorf("Fields.Project = %q, want %q", entry.Fields.Project, "my-project")
	}
	if entry.Fields.Service != "my-service" {
		t.Errorf("Fields.Service = %q, want %q", entry.Fields.Service, "my-service")
	}
	if entry.Metadata.Beat != "custom-beat" {
		t.Errorf("Metadata.Beat = %q, want %q", entry.Metadata.Beat, "custom-beat")
	}
}

func TestWriteEntry(t *testing.T) {
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

	c := New(Options{
		Project:  "demo",
		Logstash: ln.Addr().String(),
	})
	defer c.Close()

	entry := c.BuildEntry(0)
	entry.Message = "test message"
	entry.Level = "info"

	c.WriteEntry(entry)

	select {
	case data := <-received:
		trimmed := strings.TrimSpace(string(data))
		var result map[string]any
		if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
			t.Fatalf("json unmarshal error: %v (raw: %q)", err, string(data))
		}
		if result["message"] != "test message" {
			t.Errorf("message = %v, want 'test message'", result["message"])
		}
		if result["level"] != "info" {
			t.Errorf("level = %v, want 'info'", result["level"])
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for log entry")
	}
}

func TestWriteEntryWithError(t *testing.T) {
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

	c := New(Options{
		Project:  "demo",
		Logstash: ln.Addr().String(),
	})
	defer c.Close()

	entry := c.BuildEntry(0)
	entry.Message = "something failed"
	entry.Level = "error"
	entry.Error = "connection timeout"

	c.WriteEntry(entry)

	select {
	case data := <-received:
		trimmed := strings.TrimSpace(string(data))
		var result map[string]any
		if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
			t.Fatalf("json unmarshal error: %v", err)
		}
		if result["error"] != "connection timeout" {
			t.Errorf("error = %v, want 'connection timeout'", result["error"])
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for log entry")
	}
}
