package pulselogrus

import (
	"runtime"

	"github.com/HaoQi-Lee/pulse"
	"github.com/sirupsen/logrus"
)

// PulseHook implements logrus.Hook and sends log entries via pulse TCP transport.
type PulseHook struct {
	core  *pulse.Core
	level logrus.Level
}

// NewHook creates a logrus.Hook that sends logs to Logstash via pulse.
func NewHook(opts pulse.Options) *PulseHook {
	level, err := logrus.ParseLevel(opts.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	return &PulseHook{
		core:  pulse.New(opts),
		level: level,
	}
}

// Levels returns the log levels this hook should be fired for.
func (h *PulseHook) Levels() []logrus.Level {
	var levels []logrus.Level
	for _, l := range logrus.AllLevels {
		if l <= h.level {
			levels = append(levels, l)
		}
	}
	return levels
}

// Fire implements logrus.Hook. Converts the logrus.Entry to a pulse LogEntry.
func (h *PulseHook) Fire(entry *logrus.Entry) error {
	e := h.core.BuildEntry(0)
	e.Message = entry.Message
	e.Level = entry.Level.String()

	// Use entry.Caller if available
	if entry.Caller != nil {
		e.LoggerName = shortPath(entry.Caller.File, entry.Caller.Line)
	}

	// Copy logrus fields to Extra
	if len(entry.Data) > 0 {
		e.Extra = make(map[string]any, len(entry.Data))
		for k, v := range entry.Data {
			e.Extra[k] = v
		}
	}

	h.core.WriteEntry(e)
	return nil
}

// Close flushes the buffer and closes the TCP connection.
func (h *PulseHook) Close() {
	h.core.Close()
}

// shortPath returns the last two path segments with line number.
func shortPath(file string, line int) string {
	n := 0
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' || file[i] == '\\' {
			n++
			if n >= 2 {
				file = file[i+1:]
				break
			}
		}
	}
	return file
}

// callerFile extracts the file from runtime frames at the given skip depth.
// This is used when entry.Caller is not available.
func callerFile(skip int) string {
	_, file, _, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	return shortPath(file, 0)
}
