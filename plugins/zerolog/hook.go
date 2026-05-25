package pulsezerolog

import (
	"github.com/HaoQi-Lee/pulse"
	"github.com/rs/zerolog"
)

// PulseHook implements zerolog.Hook and sends log entries via pulse TCP transport.
type PulseHook struct {
	core *pulse.Core
}

// NewHook creates a zerolog.Hook that sends logs to Logstash via pulse.
func NewHook(opts pulse.Options) *PulseHook {
	return &PulseHook{
		core: pulse.New(opts),
	}
}

// Run implements zerolog.Hook. Called for every log event.
func (h *PulseHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	entry := h.core.BuildEntry(6) // skip through zerolog internals to reach user code
	entry.Message = msg
	entry.Level = mapZeroLevel(level)

	h.core.WriteEntry(entry)
}

// Close flushes the buffer and closes the TCP connection.
func (h *PulseHook) Close() {
	h.core.Close()
}

func mapZeroLevel(level zerolog.Level) string {
	switch level {
	case zerolog.DebugLevel:
		return "debug"
	case zerolog.WarnLevel:
		return "warn"
	case zerolog.ErrorLevel:
		return "error"
	case zerolog.FatalLevel, zerolog.PanicLevel:
		return "error"
	default:
		return "info"
	}
}
