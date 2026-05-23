package pulse

// Options configures the pulse core.
type Options struct {
	Project    string // project name
	Logstash   string // Logstash TCP address, e.g. "a.b.c.d:4560"
	Service    string // default: "golang"
	Beat       string // default: "logback"
	Level      string // default: "info", options: debug/info/warn/error
	BufferSize int    // default: 1024
}

func (o *Options) applyDefaults() {
	if o.Service == "" {
		o.Service = "golang"
	}
	if o.Beat == "" {
		o.Beat = "logback"
	}
	if o.Level == "" {
		o.Level = "info"
	}
	if o.BufferSize <= 0 {
		o.BufferSize = 1024
	}
}
