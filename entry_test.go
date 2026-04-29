package pulse

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLogEntryJSONSerialization(t *testing.T) {
	entry := LogEntry{
		ThreadName: "12345",
		Host:       "server-01",
		Timestamp:  "2026-04-29T10:30:00.123Z",
		LoggerName: "main.go:15",
		Metadata:   Metadata{Beat: "logback"},
		Fields:     FieldInfo{Project: "demo", Service: "golang"},
		Message:    "服务启动",
		Level:      "info",
	}

	b, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatal(err)
	}

	if result["thread_name"] != "12345" {
		t.Errorf("thread_name = %v, want 12345", result["thread_name"])
	}
	if result["host"] != "server-01" {
		t.Errorf("host = %v, want server-01", result["host"])
	}
	if result["@timestamp"] != "2026-04-29T10:30:00.123Z" {
		t.Errorf("@timestamp = %v, want 2026-04-29T10:30:00.123Z", result["@timestamp"])
	}
	if result["logger_name"] != "main.go:15" {
		t.Errorf("logger_name = %v, want main.go:15", result["logger_name"])
	}
	if result["message"] != "服务启动" {
		t.Errorf("message = %v, want 服务启动", result["message"])
	}
	if result["level"] != "info" {
		t.Errorf("level = %v, want info", result["level"])
	}

	meta, ok := result["@metadata"].(map[string]any)
	if !ok || meta["beat"] != "logback" {
		t.Errorf("@metadata = %v, want {beat:logback}", result["@metadata"])
	}
	fields, ok := result["fields"].(map[string]any)
	if !ok || fields["project"] != "demo" || fields["service"] != "golang" {
		t.Errorf("fields = %v, want {project:demo, service:golang}", result["fields"])
	}
}

func TestLogEntryOmitEmpty(t *testing.T) {
	entry := LogEntry{
		ThreadName: "12345",
		Host:       "server-01",
		Timestamp:  "2026-04-29T10:30:00.123Z",
		LoggerName: "main.go:15",
		Metadata:   Metadata{Beat: "logback"},
		Fields:     FieldInfo{Project: "demo", Service: "golang"},
		Message:    "hello",
	}

	b, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}

	s := string(b)
	if strings.Contains(s, `"level"`) {
		t.Error("level should be omitted when empty")
	}
	if strings.Contains(s, `"error"`) {
		t.Error("error should be omitted when empty")
	}
	if strings.Contains(s, `"extra"`) {
		t.Error("extra should be omitted when empty")
	}
}

func TestLogEntryWithExtra(t *testing.T) {
	entry := LogEntry{
		ThreadName: "12345",
		Host:       "server-01",
		Timestamp:  "2026-04-29T10:30:00.123Z",
		LoggerName: "main.go:15",
		Metadata:   Metadata{Beat: "logback"},
		Fields:     FieldInfo{Project: "demo", Service: "golang"},
		Message:    "hello",
		Level:      "error",
		Error:      "connection timeout",
		Extra: map[string]any{
			"request_id": "abc-123",
			"retry":      float64(3),
		},
	}

	b, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatal(err)
	}

	if result["level"] != "error" {
		t.Errorf("level = %v, want error", result["level"])
	}
	if result["error"] != "connection timeout" {
		t.Errorf("error = %v, want connection timeout", result["error"])
	}
	extra := result["extra"].(map[string]any)
	if extra["request_id"] != "abc-123" {
		t.Errorf("extra.request_id = %v, want abc-123", extra["request_id"])
	}
	if extra["retry"] != float64(3) {
		t.Errorf("extra.retry = %v, want 3", extra["retry"])
	}
}
