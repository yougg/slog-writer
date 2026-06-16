package slogw

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestGoIDHandlerPreservesWrapper(t *testing.T) {
	handler := &goIDHandler{
		handler: slog.NewTextHandler(io.Discard, nil),
	}

	if _, ok := handler.WithAttrs([]slog.Attr{slog.String("key", "value")}).(*goIDHandler); !ok {
		t.Fatal("WithAttrs should preserve goIDHandler wrapper")
	}

	if _, ok := handler.WithGroup("group").(*goIDHandler); !ok {
		t.Fatal("WithGroup should preserve goIDHandler wrapper")
	}
}

func TestNew(t *testing.T) {
	type args struct {
		file       string
		level      string
		maxSize    int64
		maxBackups int
	}
	tests := []struct {
		name string
		args args
		want *slog.Logger
	}{
		{
			name: "test_log",
			args: args{
				file:       "test.log",
				level:      LevelInfo,
				maxSize:    1024,
				maxBackups: 3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.args.file,
				WithLevel(tt.args.level),
				WithMaxSize(tt.args.maxSize),
				WithMaxBackups(tt.args.maxBackups),
			)
			logger.Debug("debug log message")
			logger.Info("info log", `case`, tt.name, `file`, tt.args.file, `level`, tt.args.level)
			logger.Warn("warning message")
			logger.Error("error......")
		})
	}
}

func TestSetDefault(t *testing.T) {
	type args struct {
		file       string
		level      string
		maxSize    int64
		maxBackups int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "default_log",
			args: args{
				file:       "default.log",
				level:      LevelDebug,
				maxSize:    1024,
				maxBackups: 3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetDefault(tt.args.file,
				WithLevel(tt.args.level),
				WithMaxSize(tt.args.maxSize),
				WithMaxBackups(tt.args.maxBackups),
			)
			slog.Debug("debug log", `case`, tt.name, `file`, tt.args.file, `level`, tt.args.level)
			slog.Info("information message")
			slog.Warn("warning message")
			slog.Error("error......")
		})
	}
}

func TestNewWithOptions(t *testing.T) {
	t.Run("FormatText", func(t *testing.T) {
		logger := New("test_text.log",
			WithLevel(LevelInfo),
			WithMaxSize(1024),
			WithMaxBackups(3),
			WithFormat(FormatText),
		)
		logger.Info("text format log")
	})

	t.Run("FormatJSON", func(t *testing.T) {
		logger := New("test_json.log",
			WithLevel(LevelInfo),
			WithMaxSize(1024),
			WithMaxBackups(3),
			WithFormat(FormatJSON),
		)
		logger.Info("json format log")
	})

	t.Run("WithStack", func(t *testing.T) {
		logger := New("test_stack.log",
			WithLevel(LevelInfo),
			WithMaxSize(1024),
			WithMaxBackups(3),
			WithFormat(FormatJSON),
			WithStack(true),
		)
		logger.Info("stack trace log")
	})
}

func TestTakeDuplicate(t *testing.T) {
	for range 5 {
		stackStr := Take(0)
		count := strings.Count(stackStr, "TestTakeDuplicate")
		if count > 1 {
			t.Fatalf("Take() return with duplicates, count: %d, stack: %s", count, stackStr)
		}
	}
}

func TestDynamicConfiguration(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "dynamic_test.log")

	// 1. Default configuration: level=info, addSource=false, stack=false
	logger := New(tempFile,
		WithLevel(LevelInfo),
		WithAddSource(false),
		WithStack(false),
		WithFormat(FormatJSON),
	)

	logger.Debug("debug msg 1") // Log level is info, should not be output
	logger.Info("info msg 1")   // Should be output

	// 2. Dynamically adjust log level to debug
	SetLoggerLevel(logger, LevelDebug)
	logger.Debug("debug msg 2") // Should be output

	// 3. Dynamically adjust AddSource to true
	SetLoggerAddSource(logger, true)
	logger.Info("info msg 2") // Should be output and contain source field

	// 4. Dynamically adjust stackTrace to true
	SetLoggerStackTrace(logger, true)
	logger.Info("info msg 3") // Should be output and contain stack field

	// 5. Verify that the derived Logger also shares this modification
	subLogger := logger.With("subKey", "subVal")
	subLogger.Info("info subMsg") // Should be output and contain subKey, source, and stack fields

	// 6. Dynamically restore to fully closed state, and set log level to warn
	SetLoggerLevel(logger, LevelWarn)
	SetLoggerAddSource(logger, false)
	SetLoggerStackTrace(logger, false)

	logger.Info("info msg 4") // Should not be output (log level is warn)
	logger.Warn("warn msg 4") // Should be output and not contain source and stack fields

	// Verify the content of the log file
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("failed to read test log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var logs []map[string]any
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var logData map[string]any
		if err := json.Unmarshal([]byte(line), &logData); err != nil {
			t.Fatalf("failed to unmarshal log line %q: %v", line, err)
		}
		logs = append(logs, logData)
	}

	if len(logs) != 6 {
		t.Fatalf("expected 6 logs, got %d", len(logs))
	}

	// 1. Info message 1
	if logs[0]["msg"] != "info msg 1" {
		t.Errorf("log 0 msg mismatch: %v", logs[0]["msg"])
	}
	if _, ok := logs[0]["source"]; ok {
		t.Errorf("log 0 should not contain source")
	}
	if _, ok := logs[0]["stack"]; ok {
		t.Errorf("log 0 should not contain stack")
	}

	// 2. Debug message 2
	if logs[1]["msg"] != "debug msg 2" {
		t.Errorf("log 1 msg mismatch: %v", logs[1]["msg"])
	}

	// 3. Info message 2
	if logs[2]["msg"] != "info msg 2" {
		t.Errorf("log 2 msg mismatch: %v", logs[2]["msg"])
	}
	if _, ok := logs[2]["source"]; !ok {
		t.Errorf("log 2 should contain source")
	}

	// 4. Info message 3
	if logs[3]["msg"] != "info msg 3" {
		t.Errorf("log 3 msg mismatch: %v", logs[3]["msg"])
	}
	if _, ok := logs[3]["stack"]; !ok {
		t.Errorf("log 3 should contain stack")
	}

	// 5. Derived info message
	if logs[4]["msg"] != "info subMsg" {
		t.Errorf("log 4 msg mismatch: %v", logs[4]["msg"])
	}
	if logs[4]["subKey"] != "subVal" {
		t.Errorf("log 4 subKey mismatch: %v", logs[4]["subKey"])
	}
	if _, ok := logs[4]["source"]; !ok {
		t.Errorf("log 4 should contain source")
	}
	if _, ok := logs[4]["stack"]; !ok {
		t.Errorf("log 4 should contain stack")
	}

	// 6. Warning message 4
	if logs[5]["msg"] != "warn msg 4" {
		t.Errorf("log 5 msg mismatch: %v", logs[5]["msg"])
	}
	if _, ok := logs[5]["source"]; ok {
		t.Errorf("log 5 should not contain source")
	}
	if _, ok := logs[5]["stack"]; ok {
		t.Errorf("log 5 should not contain stack")
	}
}

func TestDefaultLoggerDynamicConfiguration(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "default_dynamic_test.log")

	SetDefault(tempFile,
		WithLevel(LevelInfo),
		WithAddSource(false),
		WithStack(false),
		WithFormat(FormatJSON),
	)

	slog.Debug("default debug msg 1") // Should not be output
	slog.Info("default info msg 1")   // Should be output

	SetLevel(LevelDebug)
	slog.Debug("default debug msg 2") // Should be output

	SetAddSource(true)
	slog.Info("default info msg 2") // Should be output and contain source field

	SetStackTrace(true)
	slog.Info("default info msg 3") // Should be output and contain stack field

	// Restore default settings
	SetLevel(LevelInfo)
	SetAddSource(false)
	SetStackTrace(false)

	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("failed to read test log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var logs []map[string]any
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var logData map[string]any
		if err := json.Unmarshal([]byte(line), &logData); err != nil {
			t.Fatalf("failed to unmarshal log line %q: %v", line, err)
		}
		logs = append(logs, logData)
	}

	if len(logs) != 4 {
		t.Fatalf("expected 4 logs, got %d", len(logs))
	}

	if logs[0]["msg"] != "default info msg 1" {
		t.Errorf("log 0 mismatch: %v", logs[0])
	}
	if logs[1]["msg"] != "default debug msg 2" {
		t.Errorf("log 1 mismatch: %v", logs[1])
	}
	if logs[2]["msg"] != "default info msg 2" {
		t.Errorf("log 2 mismatch: %v", logs[2])
	}
	if _, ok := logs[2]["source"]; !ok {
		t.Errorf("log 2 should contain source")
	}
	if logs[3]["msg"] != "default info msg 3" {
		t.Errorf("log 3 mismatch: %v", logs[3])
	}
	if _, ok := logs[3]["stack"]; !ok {
		t.Errorf("log 3 should contain stack")
	}
}

func TestDynamicConfigurationConcurrency(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "concurrency_test.log")
	logger := New(tempFile,
		WithLevel(LevelInfo),
		WithAddSource(false),
		WithStack(false),
		WithFormat(FormatJSON),
	)

	var wg sync.WaitGroup
	// Start log writing goroutines
	for i := range 10 {
		wg.Go(func() {
			for j := range 100 {
				logger.Info("concurrency msg", "goroutine", i, "seq", j)
				logger.Debug("concurrency debug msg", "goroutine", i, "seq", j)
			}
		})
	}

	// Start configuration modification goroutine
	wg.Go(func() {
		for i := range 50 {
			switch i % 3 {
			case 0:
				SetLoggerLevel(logger, LevelDebug)
				SetLoggerAddSource(logger, true)
				SetLoggerStackTrace(logger, true)
			case 1:
				SetLoggerLevel(logger, LevelInfo)
				SetLoggerAddSource(logger, false)
				SetLoggerStackTrace(logger, false)
			default:
				SetLoggerLevel(logger, LevelWarn)
				SetLoggerAddSource(logger, true)
				SetLoggerStackTrace(logger, false)
			}
		}
	})

	wg.Wait()
}

func TestDisableSymlinkRotation(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "nosymlink.log")

	// 1. Initialize a Logger with symlink disabled
	logger := New(logFile,
		WithSymlink(false),
		WithMaxSize(50),
		WithLevel(LevelInfo),
		WithFormat(FormatText),
	)

	// Write the first log message
	logger.Info("init message")

	// Verify it is indeed not a symlink
	info, err := os.Lstat(logFile)
	if err != nil {
		t.Fatalf("failed to stat log file: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("expected log file not to be a symlink")
	}

	// Write more logs to trigger multiple rotations
	for i := range 10 {
		logger.Info("excessive message to force rotation", "idx", i)
	}

	// Verify the active log file still exists and is not a symlink
	info2, err := os.Lstat(logFile)
	if err != nil {
		t.Fatalf("failed to stat active log file after rotations: %v", err)
	}
	if info2.Mode()&os.ModeSymlink != 0 {
		t.Fatal("active log file after rotations should not be a symlink")
	}

	// Verify that backup files are indeed generated in the directory
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}

	var backupCount int
	for _, f := range files {
		name := f.Name()
		if name != "nosymlink.log" && strings.HasPrefix(name, "nosymlink.") && strings.HasSuffix(name, ".log") {
			backupCount++
		}
	}
	if backupCount == 0 {
		t.Fatal("expected rotated backup files to be created, but found none")
	}

	// 2. Verify that re-initializing this Logger allows continuing to append in direct write mode
	logger2 := New(logFile,
		WithSymlink(false),
		WithMaxSize(1024),
		WithLevel(LevelInfo),
		WithFormat(FormatText),
	)
	logger2.Info("re-initialized active write")

	info3, err := os.Lstat(logFile)
	if err != nil {
		t.Fatalf("failed to stat log file after re-init: %v", err)
	}
	if info3.Mode()&os.ModeSymlink != 0 {
		t.Fatal("re-initialized log file should not be a symlink")
	}
}
