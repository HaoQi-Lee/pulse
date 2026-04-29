package pulse

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Core is the central pulse instance that manages TCP transport and log entry construction.
type Core struct {
	Writer   *TcpWriter
	opts     Options
	hostname string
	pid      string
	local    *time.Location
}

// New creates a new Core instance with the given options.
func New(opts Options) *Core {
	opts.applyDefaults()

	local, _ := time.LoadLocation("")
	hostname, _ := os.Hostname()
	pid := strconv.Itoa(os.Getpid())

	return &Core{
		Writer:   NewTcpWriter(opts.Logstash, opts.BufferSize),
		opts:     opts,
		hostname: hostname,
		pid:      pid,
		local:    local,
	}
}

// BuildEntry creates a LogEntry pre-filled with common fields.
// callerSkip is the number of stack frames to skip when capturing the caller.
func (c *Core) BuildEntry(callerSkip int) *LogEntry {
	return &LogEntry{
		ThreadName: c.pid,
		Host:       c.hostname,
		Timestamp:  time.Now().In(c.local).Format("2006-01-02T15:04:05.999Z"),
		LoggerName: GetCaller(callerSkip),
		Metadata:   Metadata{Beat: c.opts.Beat},
		Fields:     FieldInfo{Project: c.opts.Project, Service: c.opts.Service},
	}
}

// WriteEntry serializes the LogEntry to JSON and writes it to the TcpWriter.
func (c *Core) WriteEntry(entry *LogEntry) {
	b, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintln(os.Stderr, "[pulse] json marshal error:", err)
		return
	}
	b = append(b, '\n')
	c.Writer.Write(b)
}

// Close flushes the buffer and closes the TCP connection.
func (c *Core) Close() {
	c.Writer.Close()
}
