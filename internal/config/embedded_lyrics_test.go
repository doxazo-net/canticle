package config

import "testing"

func TestLoad_EmbeddedLyrics(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	tests := []struct {
		env  string
		want string
	}{
		{"extract", "extract"},
		{"RESPECT", "respect"}, // normalized to lowercase
		{"respect", "respect"},
		{"bogus", "off"}, // invalid clamps to off
		{"", "off"},      // unset defaults to off
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv("MXLRC_EMBEDDED_LYRICS", tt.env)
			cfg, err := Load("")
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.Output.EmbeddedLyrics != tt.want {
				t.Fatalf("env %q -> %q; want %q", tt.env, cfg.Output.EmbeddedLyrics, tt.want)
			}
		})
	}
}
