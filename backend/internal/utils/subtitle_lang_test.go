package utils

import "testing"

func TestExtractSubtitleLanguageFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"Heart of the Beast.id.srt", "Indonesian"},
		{"Heart of the Beast.ind.srt", "Indonesian"},
		{"Heart of the Beast_indo.srt", "Indonesian"},
		{"Heart of the Beast.id.forced.srt", "Indonesian"},
		{"Movie.en.sdh.srt", "English"},
		{"It Follows.srt", "Unknown"},
		{"Heart of the Beast.srt", "Unknown"},
	}
	for _, tc := range tests {
		if got := ExtractSubtitleLanguageFromFilename(tc.filename); got != tc.want {
			t.Errorf("ExtractSubtitleLanguageFromFilename(%q) = %q, want %q", tc.filename, got, tc.want)
		}
	}
}

func TestStripSubtitleLanguageSuffix(t *testing.T) {
	tests := []struct {
		base string
		want string
	}{
		{"Heart of the Beast.id", "Heart of the Beast"},
		{"Heart of the Beast.id.forced", "Heart of the Beast"},
		{"Heart of the Beast", "Heart of the Beast"},
	}
	for _, tc := range tests {
		if got := StripSubtitleLanguageSuffix(tc.base); got != tc.want {
			t.Errorf("StripSubtitleLanguageSuffix(%q) = %q, want %q", tc.base, got, tc.want)
		}
	}
}
