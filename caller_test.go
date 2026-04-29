package pulse

import (
	"strings"
	"testing"
)

func TestGetCaller(t *testing.T) {
	result := GetCaller(0)
	if result == "" {
		t.Fatal("GetCaller returned empty string")
	}

	count := strings.Count(result, "/") + strings.Count(result, "\\")
	if count != 1 {
		t.Errorf("GetCaller = %q, expected exactly 1 separator, got %d", result, count)
	}

	if !strings.HasSuffix(result, "caller_test.go") {
		t.Errorf("GetCaller = %q, expected to end with caller_test.go", result)
	}
}

func TestGetCallerSkip(t *testing.T) {
	result := helperCaller()
	if result == "" {
		t.Fatal("GetCaller with skip returned empty string")
	}
	// helperCaller calls GetCaller(1), so result should be this test file
	if !strings.HasSuffix(result, "caller_test.go") {
		t.Errorf("GetCaller(skip=1) = %q, expected to end with caller_test.go", result)
	}
}

func helperCaller() string {
	return GetCaller(0)
}
