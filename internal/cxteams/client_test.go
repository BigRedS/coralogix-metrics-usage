package cxteams

import "testing"

func TestSanitizeForFilename(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"AcmeCorp", "AcmeCorp"},
		{"Acme Corp", "Acme_Corp"},
		{"  spaces  ", "spaces"},
		{"path/with/slashes", "path_with_slashes"},
		{"Weird*Chars!?", "Weird_Chars"},
		{"multiple   spaces", "multiple_spaces"},
		{"keep.allowed-chars_1", "keep.allowed-chars_1"},
		{"---", "---"},
		{"", ""},
		{"emoji 🤷 here", "emoji_here"},
	}
	for _, c := range cases {
		got := SanitizeForFilename(c.in)
		if got != c.want {
			t.Errorf("SanitizeForFilename(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
