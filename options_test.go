package pulse

import "testing"

func TestOptionsApplyDefaults(t *testing.T) {
	opts := Options{}
	opts.applyDefaults()

	if opts.Service != "golang" {
		t.Errorf("Service = %q, want %q", opts.Service, "golang")
	}
	if opts.Beat != "logback" {
		t.Errorf("Beat = %q, want %q", opts.Beat, "logback")
	}
	if opts.Level != "info" {
		t.Errorf("Level = %q, want %q", opts.Level, "info")
	}
	if opts.BufferSize != 1024 {
		t.Errorf("BufferSize = %d, want %d", opts.BufferSize, 1024)
	}
}

func TestOptionsCustomValues(t *testing.T) {
	opts := Options{
		Service:    "custom-service",
		Beat:       "custom-beat",
		Level:      "debug",
		BufferSize: 2048,
	}
	opts.applyDefaults()

	if opts.Service != "custom-service" {
		t.Errorf("Service = %q, want %q", opts.Service, "custom-service")
	}
	if opts.Beat != "custom-beat" {
		t.Errorf("Beat = %q, want %q", opts.Beat, "custom-beat")
	}
	if opts.Level != "debug" {
		t.Errorf("Level = %q, want %q", opts.Level, "debug")
	}
	if opts.BufferSize != 2048 {
		t.Errorf("BufferSize = %d, want %d", opts.BufferSize, 2048)
	}
}

func TestOptionsZeroBufferSize(t *testing.T) {
	opts := Options{BufferSize: 0}
	opts.applyDefaults()
	if opts.BufferSize != 1024 {
		t.Errorf("BufferSize = %d, want %d", opts.BufferSize, 1024)
	}
}
