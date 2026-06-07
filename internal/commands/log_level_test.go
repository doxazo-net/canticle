package commands

import (
	"context"
	"log/slog"
	"testing"
)

func TestApplyLogLevel(t *testing.T) {
	// Restore the default level after the test regardless of outcome.
	t.Cleanup(func() { slog.SetLogLoggerLevel(slog.LevelInfo) })

	tests := []struct {
		value     string
		wantDebug bool
		wantInfo  bool
	}{
		{"debug", true, true},
		{"warn", false, false},
		{"", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Setenv("MXLRC_LOG_LEVEL", tt.value)
			applyLogLevel()
			ctx := context.Background()
			if got := slog.Default().Enabled(ctx, slog.LevelDebug); got != tt.wantDebug {
				t.Errorf("debug enabled = %v; want %v", got, tt.wantDebug)
			}
			if got := slog.Default().Enabled(ctx, slog.LevelInfo); got != tt.wantInfo {
				t.Errorf("info enabled = %v; want %v", got, tt.wantInfo)
			}
		})
	}
}
