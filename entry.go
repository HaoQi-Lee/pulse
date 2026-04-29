package pulse

// LogEntry represents a structured log entry sent to Logstash.
type LogEntry struct {
	ThreadName string         `json:"thread_name"`
	Host       string         `json:"host"`
	Timestamp  string         `json:"@timestamp"`
	LoggerName string         `json:"logger_name"`
	Metadata   Metadata       `json:"@metadata"`
	Fields     FieldInfo      `json:"fields"`
	Message    string         `json:"message"`
	Level      string         `json:"level,omitempty"`
	Error      string         `json:"error,omitempty"`
	Extra      map[string]any `json:"extra,omitempty"`
}

// Metadata holds beat information.
type Metadata struct {
	Beat string `json:"beat"`
}

// FieldInfo holds project and service metadata.
type FieldInfo struct {
	Project string `json:"project"`
	Service string `json:"service"`
}
