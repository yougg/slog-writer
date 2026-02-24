//go:build amd64 || arm64 || arm || 386 || mipsle

package slogw

func goid() int

// GoID returns the current goroutine id.
// It exactly matches goroutine id of the stack trace.
func GoID() int {
	return goid()
}
