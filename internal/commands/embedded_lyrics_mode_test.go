package commands

import "testing"

func TestEmbeddedLyricsMode(t *testing.T) {
	respect := "respect"
	bogus := "bogus"
	upper := "EXTRACT"
	tests := []struct {
		name   string
		flag   *string
		cfgVal string
		want   string
	}{
		{"flag wins over config", &respect, "off", "respect"},
		{"config used when no flag", nil, "extract", "extract"},
		{"flag normalized", &upper, "off", "extract"},
		{"invalid flag clamps to off", &bogus, "extract", "off"},
		{"empty config defaults off", nil, "", "off"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := embeddedLyricsMode(tt.flag, tt.cfgVal); got != tt.want {
				t.Fatalf("embeddedLyricsMode(%v, %q) = %q; want %q", tt.flag, tt.cfgVal, got, tt.want)
			}
		})
	}
}
