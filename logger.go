package slogw

import (
	"context"
	"log/slog"
	"os"
)

// goIDHandler wraps given handler and add goid attr to each log content
type goIDHandler struct {
	handler    slog.Handler
	stackTrace bool
}

// Enabled reports whether the handler handles records at the given level.
func (h *goIDHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// WithAttrs returns a new Handler whose attributes consist of both the receiver's attributes and the arguments.
func (h *goIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &goIDHandler{
		handler:    h.handler.WithAttrs(attrs),
		stackTrace: h.stackTrace,
	}
}

// WithGroup returns a new Handler with the given group appended to the receiver's existing groups.
func (h *goIDHandler) WithGroup(name string) slog.Handler {
	return &goIDHandler{
		handler:    h.handler.WithGroup(name),
		stackTrace: h.stackTrace,
	}
}

// Handle rewrite standard JSON handler to add goroutine ID for each goroutine calls
func (h *goIDHandler) Handle(ctx context.Context, record slog.Record) error {
	record.AddAttrs(slog.Attr{
		Key:   `goid`,
		Value: slog.IntValue(goid()),
	})
	if h.stackTrace {
		record.AddAttrs(slog.Attr{
			Key:   `stack`,
			Value: slog.StringValue(Take(3)),
		})
	}

	err := h.handler.Handle(ctx, record)
	if record.Level == FatalLevel {
		os.Exit(1)
	}

	return err
}

const (
	TraceLevel = slog.Level(-8)
	DebugLevel = slog.LevelDebug
	InfoLevel  = slog.LevelInfo
	WarnLevel  = slog.LevelWarn
	ErrorLevel = slog.LevelError
	FatalLevel = slog.Level(12)

	LevelTrace = "trace"
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
	LevelFatal = "fatal"
)

var (
	Levels = map[string]slog.Level{
		LevelTrace: TraceLevel,
		LevelDebug: DebugLevel,
		LevelInfo:  InfoLevel,
		LevelWarn:  WarnLevel,
		LevelError: ErrorLevel,
		LevelFatal: FatalLevel,
	}
)

// Format represents the log output format.
type Format byte

const (
	FormatJSON Format = iota // JSON format
	FormatText               // Text format
)

// options contains all optional configurations for creating a Logger.
type options struct {
	slog.HandlerOptions
	format     Format
	level      string
	maxSize    int64
	maxBackups int
	stackTrace bool
}

// Option defines the function type to configure options.
type Option func(*options)

// WithLevel sets the log level (e.g., trace, debug, info, warn, error, fatal).
func WithLevel(level string) Option {
	return func(o *options) {
		o.level = level
	}
}

// WithMaxSize sets the maximum size in bytes of the log file before it gets rotated.
func WithMaxSize(size int64) Option {
	return func(o *options) {
		o.maxSize = size
	}
}

// WithMaxBackups sets the maximum number of old log files to retain.
func WithMaxBackups(backups int) Option {
	return func(o *options) {
		o.maxBackups = backups
	}
}

// WithFormat sets the log output format (FormatJSON, FormatText).
func WithFormat(f Format) Option {
	return func(o *options) {
		o.format = f
	}
}

// WithAddSource sets whether to record the source file and line number in the log.
func WithAddSource(addSource bool) Option {
	return func(o *options) {
		o.AddSource = addSource
	}
}

// WithReplaceAttr sets the function to rewrite attributes.
func WithReplaceAttr(replaceAttr func(groups []string, a slog.Attr) slog.Attr) Option {
	return func(o *options) {
		o.ReplaceAttr = replaceAttr
	}
}

// WithStack sets whether to record stacktrace information in the log.
func WithStack(stacktrace bool) Option {
	return func(o *options) {
		o.stackTrace = stacktrace
	}
}

// New create new file logger
//
//	file: log file path
//	opts: functional options to configure the logger
func New(file string, opts ...Option) *slog.Logger {
	o := &options{
		format: FormatText,
		level:  LevelInfo,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}

	if o.Level == nil {
		o.Level = Levels[o.level]
	}

	writer := &FileWriter{
		EnsureFolder: true,
		Filename:     file,
		MaxBackups:   o.maxBackups,
		MaxSize:      o.maxSize,
		LocalTime:    true,
	}

	var handler slog.Handler
	if o.format == FormatText {
		handler = slog.NewTextHandler(writer, &o.HandlerOptions)
	} else {
		handler = slog.NewJSONHandler(writer, &o.HandlerOptions)
	}

	return slog.New(&goIDHandler{
		handler:    handler,
		stackTrace: o.stackTrace,
	})
}

// SetDefault set global default logger
//
//	file: log file path
//	opts: functional options to configure the logger
func SetDefault(file string, opts ...Option) {
	slog.SetDefault(New(file, opts...))
}
