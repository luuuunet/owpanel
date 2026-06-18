package aisite

import "testing"

func TestParsePHPVersionRequired(t *testing.T) {
	cases := map[string]string{
		`{"require":{"php":"^8.4.1"}}`:           "8.4",
		`{"require":{"php":">=8.2.0"}}`:          "8.2",
		`{"require":{"php":"^8.3||^8.4"}}`:       "8.3",
		`{"require":{"php":"^7.4|^8.0"}}`:        "7.4",
	}
	for input, want := range cases {
		got := parsePHPVersionRequired(input)
		if got != want {
			t.Fatalf("parsePHPVersionRequired(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSuggestedNodeAppKey(t *testing.T) {
	snap := &RepoSnapshot{NodeMajorRequired: 18}
	if got := snap.suggestedNodeAppKey(); got != "nodejs18" {
		t.Fatalf("got %s", got)
	}
}
