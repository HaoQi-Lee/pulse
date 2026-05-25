package pulsezap

import (
	"fmt"

	"github.com/HaoQi-Lee/pulse"
	"go.uber.org/zap/zapcore"
)

// PulseCore implements zapcore.Core and sends log entries via pulse TCP transport.
type PulseCore struct {
	core   *pulse.Core
	level  zapcore.Level
	fields []zapcore.Field
}

// NewCore creates a zapcore.Core that sends logs to Logstash via pulse.
func NewCore(opts pulse.Options) *PulseCore {
	level := parseZapLevel(opts.Level)
	return &PulseCore{
		core:  pulse.New(opts),
		level: level,
	}
}

func parseZapLevel(s string) zapcore.Level {
	switch s {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// Enabled returns true if the given level should be logged.
func (c *PulseCore) Enabled(level zapcore.Level) bool {
	return level >= c.level
}

// With returns a new core with the given fields.
func (c *PulseCore) With(fields []zapcore.Field) zapcore.Core {
	newFields := make([]zapcore.Field, len(c.fields)+len(fields))
	copy(newFields, c.fields)
	copy(newFields[len(c.fields):], fields)
	return &PulseCore{
		core:   c.core,
		level:  c.level,
		fields: newFields,
	}
}

// Check adds this core to the CheckedEntry if the level is enabled.
func (c *PulseCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return ce
}

// Write processes the zap entry and sends it via pulse.
func (c *PulseCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	e := c.core.BuildEntry(0)
	e.Message = entry.Message
	e.Level = mapZapLevel(entry.Level)

	// Use zap's caller info if available
	if entry.Caller.Defined {
		e.LoggerName = fmt.Sprintf("%s:%d", shortPath(entry.Caller.File), entry.Caller.Line)
	}

	// Collect all fields into Extra
	encoder := zapcore.NewMapObjectEncoder()
	for _, f := range c.fields {
		f.AddTo(encoder)
	}
	for _, f := range fields {
		f.AddTo(encoder)
	}
	if len(encoder.Fields) > 0 {
		e.Extra = encoder.Fields
	}

	c.core.WriteEntry(e)
	return nil
}

// Sync is a no-op for pulse (async writes).
func (c *PulseCore) Sync() error {
	return nil
}

// Close flushes the buffer and closes the TCP connection.
func (c *PulseCore) Close() {
	c.core.Close()
}

func mapZapLevel(level zapcore.Level) string {
	switch level {
	case zapcore.DebugLevel:
		return "debug"
	case zapcore.WarnLevel:
		return "warn"
	case zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		return "error"
	default:
		return "info"
	}
}

func shortPath(file string) string {
	n := 0
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' || file[i] == '\\' {
			n++
			if n >= 2 {
				return file[i+1:]
			}
		}
	}
	return file
}
