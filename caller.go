package pulse

import "runtime"

// GetCaller returns the file path of the caller at the given skip depth.
// The result contains the last two path segments, e.g. "pulse/logger.go".
func GetCaller(skip int) string {
	_, file, _, ok := runtime.Caller(skip + 1)
	if !ok {
		return ""
	}
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
