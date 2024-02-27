//go:build amd64 || arm64

package slogw

func goid() int

// GoID returns the current goroutine id.
// It exactly matches goroutine id of the stack trace.
func GoID() int {
	return goid()
}
