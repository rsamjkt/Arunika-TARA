package updater

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v2.2.0", "v2.1.2", true},
		{"v2.2.0-mahatma", "v2.1.2-mahatma", true},
		{"v2.1.2-mahatma", "v2.1.2-mahatma", false},
		{"v2.1.2", "v2.1.3", false},
		{"v3.0.0-mahatma", "v2.99.99", true},
		{"v2.1.10", "v2.1.9", true},
		{"v2.1.9", "v2.1.10", false},
		// Invalid → false
		{"latest", "v1.0.0", false},
		{"v2.0", "v1.0.0", false},
	}
	for _, c := range cases {
		got := isNewer(c.latest, c.current)
		if got != c.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestMatchAsset(t *testing.T) {
	cases := []struct {
		name, pattern string
		want          bool
	}{
		{"apm-windows-amd64.exe", "apm-windows-amd64.exe", true},
		{"apm-windows-amd64.exe", "APM-Windows-AMD64.exe", true}, // case-insensitive
		{"apm-windows-amd64.exe", "apm-*-amd64.exe", true},
		{"apm-windows-amd64.exe", "apm-*.exe", true},
		{"apm-mac-universal.app.zip", "apm-windows-*", false},
		{"apm-windows-amd64.exe", "*", true},
		{"foo.exe", "", false},
	}
	for _, c := range cases {
		got := matchAsset(c.name, c.pattern)
		if got != c.want {
			t.Errorf("matchAsset(%q, %q) = %v, want %v", c.name, c.pattern, got, c.want)
		}
	}
}

func TestParseVersion(t *testing.T) {
	cases := []struct {
		in   string
		want []int
	}{
		{"v2.1.2", []int{2, 1, 2}},
		{"2.1.2", []int{2, 1, 2}},
		{"v2.1.2-mahatma", []int{2, 1, 2}},
		{"v10.20.30-rc1", []int{10, 20, 30}},
		{"v2.1", nil},
		{"latest", nil},
	}
	for _, c := range cases {
		got := parseVersion(c.in)
		if len(got) != len(c.want) {
			t.Errorf("parseVersion(%q) len = %d, want %d", c.in, len(got), len(c.want))
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("parseVersion(%q)[%d] = %d, want %d", c.in, i, got[i], c.want[i])
			}
		}
	}
}
