package slogw

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileWriterReusesSymlinkTargetUntilMaxSize(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "app.log")

	writer := func() *FileWriter {
		return &FileWriter{
			Filename:   filename,
			MaxSize:    10,
			MaxBackups: 10,
			TimeFormat: "20060102150405.000000000",
		}
	}

	w := writer()
	if _, err := w.Write([]byte("12345")); err != nil {
		t.Fatalf("first write failed: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("first close failed: %v", err)
	}

	firstTarget := readLink(t, filename)
	if got := countLogFiles(t, dir); got != 1 {
		t.Fatalf("initial write created %d log files, want 1", got)
	}

	w = writer()
	if _, err := w.Write([]byte("67890")); err != nil {
		t.Fatalf("second write failed: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}

	if got := readLink(t, filename); got != firstTarget {
		t.Fatalf("symlink target changed before max size was exceeded: got %q, want %q", got, firstTarget)
	}
	if got := countLogFiles(t, dir); got != 1 {
		t.Fatalf("restart before rotation created %d log files, want 1", got)
	}

	time.Sleep(time.Millisecond)

	w = writer()
	if _, err := w.Write([]byte("x")); err != nil {
		t.Fatalf("third write failed: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("third close failed: %v", err)
	}

	waitForLinkChange(t, filename, firstTarget)
	if got := countLogFiles(t, dir); got != 2 {
		t.Fatalf("rotation created %d log files, want 2", got)
	}
}

func TestFileWriterRotatesOversizedSymlinkTargetOnStartup(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "app.log")
	oldTarget := "app.old.log"
	oldPath := filepath.Join(dir, oldTarget)

	if err := os.WriteFile(oldPath, []byte("12345678901"), 0644); err != nil {
		t.Fatalf("write old target failed: %v", err)
	}
	if err := os.Symlink(oldTarget, filename); err != nil {
		t.Fatalf("create symlink failed: %v", err)
	}

	time.Sleep(time.Millisecond)

	w := &FileWriter{
		Filename:   filename,
		MaxSize:    10,
		MaxBackups: 10,
		TimeFormat: "20060102150405.000000000",
	}
	if _, err := w.Write([]byte("x")); err != nil {
		t.Fatalf("write after oversized startup target failed: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	if got := readLink(t, filename); got == oldTarget {
		t.Fatalf("symlink still points to oversized target %q", oldTarget)
	}
	if got := countLogFiles(t, dir); got != 2 {
		t.Fatalf("startup rotation left %d log files, want 2", got)
	}
}

func readLink(t *testing.T, filename string) string {
	t.Helper()

	target, err := os.Readlink(filename)
	if err != nil {
		t.Fatalf("read symlink failed: %v", err)
	}
	return target
}

func countLogFiles(t *testing.T, dir string) int {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(dir, "app.*.log"))
	if err != nil {
		t.Fatalf("glob log files failed: %v", err)
	}
	return len(matches)
}

func waitForLinkChange(t *testing.T, filename, oldTarget string) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if target := readLink(t, filename); target != oldTarget {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("symlink target did not change from %q", oldTarget)
}
