package commands

import "testing"

// TestResolveDetectOverride covers the scan --detect-instrumental/--no-detect-instrumental
// mutual-exclusion resolution into a tri-state *bool (nil = no override).
func TestResolveDetectOverride(t *testing.T) {
	cases := []struct {
		name      string
		detect    bool
		noDetect  bool
		wantNil   bool
		wantVal   bool
		wantError bool
	}{
		{name: "neither set -> nil", wantNil: true},
		{name: "detect -> true", detect: true, wantVal: true},
		{name: "no-detect -> false", noDetect: true, wantVal: false},
		{name: "both set -> error", detect: true, noDetect: true, wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveDetectOverride(tc.detect, tc.noDetect)
			if tc.wantError {
				if err == nil {
					t.Fatal("expected error for conflicting flags; got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantNil {
				if got != nil {
					t.Errorf("override = %v; want nil", *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("override = nil; want %v", tc.wantVal)
			}
			if *got != tc.wantVal {
				t.Errorf("override = %v; want %v", *got, tc.wantVal)
			}
		})
	}
}
