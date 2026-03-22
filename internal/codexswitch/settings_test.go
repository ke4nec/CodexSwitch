package codexswitch

import (
	"os"
	"runtime"
	"testing"
)

func TestDefaultCodexHomeBasePrefersWindowsProfileEnv(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", `C:\Users\alice`)
	t.Setenv("HOMEDRIVE", "D:")
	t.Setenv("HOMEPATH", `\fallback`)

	if got := defaultCodexHomeBase("windows"); got != `C:\Users\alice` {
		t.Fatalf("expected USERPROFILE to win on windows, got %q", got)
	}
}

func TestDefaultCodexHomeBaseFallsBackToHomeDriveAndPathOnWindows(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("HOMEDRIVE", "D:")
	t.Setenv("HOMEPATH", `\workspace\alice`)

	if got := defaultCodexHomeBase("windows"); got != `D:\workspace\alice` {
		t.Fatalf("expected HOMEDRIVE/HOMEPATH fallback on windows, got %q", got)
	}
}

func TestDefaultCodexHomeBaseUsesHomeOnUnixLikePlatforms(t *testing.T) {
	t.Setenv("HOME", "/Users/alice")
	t.Setenv("USERPROFILE", `C:\Users\alice`)
	t.Setenv("HOMEDRIVE", "D:")
	t.Setenv("HOMEPATH", `\fallback`)

	if got := defaultCodexHomeBase("darwin"); got != "/Users/alice" {
		t.Fatalf("expected HOME on darwin, got %q", got)
	}

	if got := defaultCodexHomeBase("linux"); got != "/Users/alice" {
		t.Fatalf("expected HOME on linux, got %q", got)
	}
}

func TestDefaultCodexHomeBaseFallsBackToUserHomeDirWhenEnvIsMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows UserHomeDir depends on USERPROFILE/HOMEDRIVE/HOMEPATH when env is cleared in-process")
	}

	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")

	want, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir failed: %v", err)
	}

	if got := defaultCodexHomeBase("linux"); got != want {
		t.Fatalf("expected UserHomeDir fallback, want %q got %q", want, got)
	}
}
