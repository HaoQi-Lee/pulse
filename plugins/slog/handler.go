package pulseslog

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"

	"github.com/HaoQi-Lee/pulse"
)

// PulseHandler implements slog.Handler and sends log entries via pulse TCP transport.
type PulseHandler struct {
	core   *pulse.Core
	level  slog.Level
	attrs  []slog.Attr
	groups []string
}

// NewHandler creates a slog.Handler that sends logs to Logstash via pulse.
func NewHandler(opts pulse.Options) *PulseHandler {
	level := parseLevel(opts.Level)
	return &PulseHandler{
		core:  pulse.New(opts),
		level: level,
	}
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Enabled returns true if the given level should be logged.
func (h *PulseHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle processes the slog.Record and sends it via pulse.
func (h *PulseHandler) Handle(_ context.Context, r slog.Record) error {
	e := h.core.BuildEntry(0)
	e.Message = r.Message
	e.Level = mapLevel(r.Level)

	// Use PC for caller info
	if r.PC != 0 {
		frames := runtime.CallersFrames([]uintptr{r.PC})
		if frame, more := frames.Next(); more || frame.File != "" {
			e.LoggerName = fmt.Sprintf("%s:%d", shortPath(frame.File), frame.Line)
		}
	}

	// Collect attrs from handler and record
	extra := make(map[string]any)
	for _, a := range h.attrs {
		extra[a.Key] = a.Value.Resolve().Any()
	}
	r.Attrs(func(a slog.Attr) bool {
		extra[a.Key] = a.Value.Resolve().Any()
		return true
	})
	if len(extra) > 0 {
		e.Extra = extra
	}

	h.core.WriteEntry(e)
	return nil
}

// WithAttrs returns a new handler with the given attributes.
func (h *PulseHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)
	return &PulseHandler{
		core:   h.core,
		level:  h.level,
		attrs:  newAttrs,
		groups: h.groups,
	}
}

// WithGroup returns a new handler with the given group.
func (h *PulseHandler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name
	return &PulseHandler{
		core:   h.core,
		level:  h.level,
		attrs:  h.attrs,
		groups: newGroups,
	}
}

// Close flushes the buffer and closes the TCP connection.
func (h *PulseHandler) Close() {
	h.core.Close()
}

func mapLevel(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "error"
	case level >= slog.LevelWarn:
		return "warn"
	case level >= slog.LevelInfo:
		return "info"
	default:
		return "debug"
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
