package slogw

import (
	"context"
	"log/slog"
)

// goIDHandler wraps given handler and add goid attr to each log content
type goIDHandler struct {
	handler    slog.Handler
	stacktrace bool
}

// Enabled reports whether the handler handles records at the given level.
func (h *goIDHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// WithAttrs returns a new Handler whose attributes consist of both the receiver's attributes and the arguments.
func (h *goIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.handler.WithAttrs(attrs)
}

// WithGroup returns a new Handler with the given group appended to the receiver's existing groups.
func (h *goIDHandler) WithGroup(name string) slog.Handler {
	return h.handler.WithGroup(name)
}

// Handle rewrite standard json handler to add goroutine ID for each goroutine calls
func (h *goIDHandler) Handle(ctx context.Context, record slog.Record) error {
	record.AddAttrs(slog.Attr{
		Key:   `goid`,
		Value: slog.IntValue(goid()),
	})
	if h.stacktrace {
		record.AddAttrs(slog.Attr{
			Key:   `stack`,
			Value: slog.StringValue(Take(3)),
		})
	}
	return h.handler.Handle(ctx, record)
}

var (
	Levels = map[string]slog.Level{
		`debug`: slog.LevelDebug,
		`info`:  slog.LevelInfo,
		`warn`:  slog.LevelWarn,
		`error`: slog.LevelError,
	}
)

// New create new file logger
//
//	file: log file path
//	level: log level: debug, info, warn, error
//	maxSize: the maximum size in bytes of the log file before it gets rotated
//	maxBackups: the maximum number of old log files to retain
func New(file, level string, maxSize int64, maxBackups int) *slog.Logger {
	writer := &FileWriter{
		EnsureFolder: true,
		Filename:     file,
		MaxBackups:   maxBackups,
		MaxSize:      maxSize,
		LocalTime:    true,
	}
	return slog.New(&goIDHandler{
		handler: slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level:     Levels[level],
			AddSource: true,
		}),
	})
}

// NewWithStack create new file logger
//
//	file: log file path
//	level: log level: debug, info, warn, error
//	maxSize: the maximum size in bytes of the log file before it gets rotated
//	maxBackups: the maximum number of old log files to retain
func NewWithStack(file, level string, maxSize int64, maxBackups int) *slog.Logger {
	writer := &FileWriter{
		EnsureFolder: true,
		Filename:     file,
		MaxBackups:   maxBackups,
		MaxSize:      maxSize,
		LocalTime:    true,
	}
	return slog.New(&goIDHandler{
		handler: slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level: Levels[level],
		}),
		stacktrace: true,
	})
}

// SetDefault set global default logger
//
//	file: log file path
//	level: log level: debug, info, warn, error
//	maxSize: the maximum size in bytes of the log file before it gets rotated
//	maxBackups: the maximum number of old log files to retain
func SetDefault(file, level string, maxSize int64, maxBackups int) {
	slog.SetDefault(New(file, level, maxSize, maxBackups))
}

// SetDefaultWithStack set global default logger
//
//	file: log file path
//	level: log level: debug, info, warn, error
//	maxSize: the maximum size in bytes of the log file before it gets rotated
//	maxBackups: the maximum number of old log files to retain
func SetDefaultWithStack(file, level string, maxSize int64, maxBackups int) {
	slog.SetDefault(New(file, level, maxSize, maxBackups))
}
