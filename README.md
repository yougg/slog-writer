# slog-writer
File writer for Go std "log/slog" package

## functions

- add goroutine ID to each log record
- rotate log file with limit file size
- cleanup when over backup log file count

## usage

```go
package main

import (
	"context"
	"io"
	"log/slog"
	"os"

	slogw "github.com/yougg/slog-writer"
)

func main() {
	defaultLog()
	newLog()
	multiLog()
}

func defaultLog() {
	slogw.SetDefault("./test.log", "info", 1024*1024, 10)

	slog.Debug("debug log")
	slog.Info("information message", `key0`, "value0", `key1`, "value1")
	slog.Warn("warning message")
	slog.Error("do something", `err`, "error message")
}

func newLog() {
	logger := slogw.New("new.log", "debug", 1024, 3)

	logger.Debug("debug log")
	logger.Info("information message", `enabled`, logger.Enabled(context.Background(), slog.LevelInfo))
	logger.Warn("warning message")
	logger.Error("do something", `err`, "error message")
}

func multiLog() {
	fw := &slogw.FileWriter{
		Filename:     "file_writer.log",
		EnsureFolder: false,
		MaxBackups:   1,
		MaxSize:      1024,
		FileMode:     0644,
		TimeFormat:   slogw.TimeFormatUnix,
		LocalTime:    true,
		ProcessID:    false,
	}
	writer := io.MultiWriter(os.Stdout, fw)

	logger := slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logger.Debug("debug log message")
	logger.Info("debug log", `file`, fw.Filename)
	logger.Warn("warning message")
	logger.Error("error......")

	slog.SetDefault(logger)
	slog.Log(context.Background(), slog.LevelInfo, "multi writer for default slog")
	slog.With("keyX", "valueX").Warn("warn message with kv pair")
}
```