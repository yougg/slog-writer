package slogw

import (
	"log/slog"
	"testing"
)

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
				level:      "info",
				maxSize:    1024,
				maxBackups: 3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.args.file, tt.args.level, tt.args.maxSize, tt.args.maxBackups)
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
				level:      "debug",
				maxSize:    1024,
				maxBackups: 3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetDefault(tt.args.file, tt.args.level, tt.args.maxSize, tt.args.maxBackups)
			slog.Debug("debug log", `case`, tt.name, `file`, tt.args.file, `level`, tt.args.level)
			slog.Info("information message")
			slog.Warn("warning message")
			slog.Error("error......")
		})
	}
}
