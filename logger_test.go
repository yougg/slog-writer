package slogw

import (
	"io"
	"log/slog"
	"strings"
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
