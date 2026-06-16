package slogw

import (
	"context"
	"log/slog"
	"os"
	"sync/atomic"
)

type dynamicConfig struct {
	addSource  atomic.Bool
	stackTrace atomic.Bool
	level      *slog.LevelVar
}

// goIDHandler wraps the given handler and add goid attr to each log content
type goIDHandler struct {
	handler slog.Handler
	cfg     *dynamicConfig
}

// Enabled reports whether the handler handles records at the given level.
func (h *goIDHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// WithAttrs returns a new Handler whose attributes consist of both the receiver's attributes and the arguments.
func (h *goIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &goIDHandler{
		handler: h.handler.WithAttrs(attrs),
		cfg:     h.cfg,
	}
}

// WithGroup returns a new Handler with the given group appended to the receiver's existing groups.
func (h *goIDHandler) WithGroup(name string) slog.Handler {
	return &goIDHandler{
		handler: h.handler.WithGroup(name),
		cfg:     h.cfg,
	}
}

// Handle rewrite standard JSON handler to add goroutine ID for each goroutine calls
func (h *goIDHandler) Handle(ctx context.Context, record slog.Record) error {
	record.AddAttrs(slog.Attr{
		Key:   "goid",
		Value: slog.IntValue(goid()),
	})
	if h.cfg.stackTrace.Load() {
		record.AddAttrs(slog.Attr{
			Key:   "stack",
			Value: slog.StringValue(Take(3)),
		})
	}

	err := h.handler.Handle(ctx, record)
	if record.Level == FatalLevel {
		os.Exit(1)
	}

	return err
}

// SetAddSource dynamically adjusts whether to record the source file and line number.
func (h *goIDHandler) SetAddSource(addSource bool) {
	h.cfg.addSource.Store(addSource)
}

// SetLevel dynamically adjusts the log level.
func (h *goIDHandler) SetLevel(level string) {
	if l, ok := Levels[level]; ok {
		h.cfg.level.Set(l)
	}
}

// SetStackTrace dynamically adjusts whether to record stack trace information.
func (h *goIDHandler) SetStackTrace(stackTrace bool) {
	h.cfg.stackTrace.Store(stackTrace)
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

	cfg := &dynamicConfig{
		level: &slog.LevelVar{},
	}
	cfg.addSource.Store(o.AddSource)
	cfg.stackTrace.Store(o.stackTrace)
	cfg.level.Set(o.Level.Level())

	// Always let the underlying Handler record the Source, and filter it in ReplaceAttr to control its output.
	o.AddSource = true
	o.Level = cfg.level

	userReplaceAttr := o.ReplaceAttr
	o.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.SourceKey {
			if !cfg.addSource.Load() {
				return slog.Attr{}
			}
		}
		if userReplaceAttr != nil {
			return userReplaceAttr(groups, a)
		}
		return a
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
		handler: handler,
		cfg:     cfg,
	})
}

// SetDefault set global default logger
//
//	file: log file path
//	opts: functional options to configure the logger
func SetDefault(file string, opts ...Option) {
	slog.SetDefault(New(file, opts...))
}

// SetLoggerAddSource dynamically adjusts the AddSource configuration of the specified Logger.
func SetLoggerAddSource(logger *slog.Logger, addSource bool) {
	if logger == nil {
		return
	}
	if h, ok := logger.Handler().(*goIDHandler); ok {
		h.SetAddSource(addSource)
	}
}

// SetLoggerLevel dynamically adjusts the log level of the specified Logger.
func SetLoggerLevel(logger *slog.Logger, level string) {
	if logger == nil {
		return
	}
	if h, ok := logger.Handler().(*goIDHandler); ok {
		h.SetLevel(level)
	}
}

// SetLoggerStackTrace dynamically adjusts the stackTrace configuration of the specified Logger.
func SetLoggerStackTrace(logger *slog.Logger, stackTrace bool) {
	if logger == nil {
		return
	}
	if h, ok := logger.Handler().(*goIDHandler); ok {
		h.SetStackTrace(stackTrace)
	}
}

// SetAddSource dynamically adjusts the AddSource configuration of the default Logger.
func SetAddSource(addSource bool) {
	SetLoggerAddSource(slog.Default(), addSource)
}

// SetLevel dynamically adjusts the log level of the default Logger.
func SetLevel(level string) {
	SetLoggerLevel(slog.Default(), level)
}

// SetStackTrace dynamically adjusts the stackTrace configuration of the default Logger.
func SetStackTrace(stackTrace bool) {
	SetLoggerStackTrace(slog.Default(), stackTrace)
}
