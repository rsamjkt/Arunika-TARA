package detector

import "testing"

func TestMaskID(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"0001234567890012", "************0012"},
		{"3271234567890001", "************0001"},
		{"RM-12345", "****2345"},   // 8 chars → masked normal (4 stars + 4 last)
		{"RM-123", "***"},           // < 8 chars → fully masked
		{"", "***"},                // empty → fully masked
		{"1234567890", "******7890"}, // 10 char ID → 6 stars + 4 last
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := maskID(tt.in); got != tt.want {
				t.Errorf("maskID(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
