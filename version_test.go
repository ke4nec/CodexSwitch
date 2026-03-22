package main

import "testing"

func TestParseAppVersion(t *testing.T) {
	t.Run("uses productVersion from wails config", func(t *testing.T) {
		raw := []byte(`{"info":{"productVersion":"2.3.4"}}`)

		if got := parseAppVersion(raw); got != "2.3.4" {
			t.Fatalf("parseAppVersion() = %q, want %q", got, "2.3.4")
		}
	})

	t.Run("trims productVersion", func(t *testing.T) {
		raw := []byte(`{"info":{"productVersion":" 1.2.3 "}}`)

		if got := parseAppVersion(raw); got != "1.2.3" {
			t.Fatalf("parseAppVersion() = %q, want %q", got, "1.2.3")
		}
	})

	t.Run("falls back when config is invalid", func(t *testing.T) {
		if got := parseAppVersion([]byte(`{`)); got != "unknown" {
			t.Fatalf("parseAppVersion() = %q, want %q", got, "unknown")
		}
	})

	t.Run("falls back when productVersion is empty", func(t *testing.T) {
		raw := []byte(`{"info":{"productVersion":"   "}}`)

		if got := parseAppVersion(raw); got != "unknown" {
			t.Fatalf("parseAppVersion() = %q, want %q", got, "unknown")
		}
	})
}
