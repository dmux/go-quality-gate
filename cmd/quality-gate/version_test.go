package main

import (
	"strings"
	"testing"
)

func TestVersionInfo(t *testing.T) {
	if got := VersionInfo(); !strings.Contains(got, Version) {
		t.Errorf("expected VersionInfo to include %q, got %q", Version, got)
	}
}

func TestDetailedVersionInfo(t *testing.T) {
	got := DetailedVersionInfo()
	for _, want := range []string{Version, BuildDate, GitCommit} {
		if !strings.Contains(got, want) {
			t.Errorf("expected DetailedVersionInfo to include %q, got %q", want, got)
		}
	}
}
