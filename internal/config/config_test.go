package config

import (
	"strings"
	"testing"
	"time"
)

func TestParse_OK(t *testing.T) {
	cfg, err := Parse([]string{
		"--urls", "https://a.com, https://b.com",
		"--depth", "3",
		"--timeout", "1m",
		"--request-timeout", "5s",
		"--output", "out.json",
		"--log", "app.log",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.URLs) != 2 || cfg.URLs[0] != "https://a.com" || cfg.URLs[1] != "https://b.com" {
		t.Fatalf("URLs = %#v", cfg.URLs)
	}
	if cfg.Depth != 3 {
		t.Fatalf("Depth = %d", cfg.Depth)
	}
	if cfg.Timeout != time.Minute {
		t.Fatalf("Timeout = %v", cfg.Timeout)
	}
	if cfg.RequestTimeout != 5*time.Second {
		t.Fatalf("RequestTimeout = %v", cfg.RequestTimeout)
	}
	if cfg.Output != "out.json" {
		t.Fatalf("Output = %q", cfg.Output)
	}
	if cfg.LogFile != "app.log" {
		t.Fatalf("LogFile = %q", cfg.LogFile)
	}
}

func TestParse_Defaults(t *testing.T) {
	cfg, err := Parse([]string{"--urls", "https://example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Depth != 0 {
		t.Fatalf("Depth = %d, want 0", cfg.Depth)
	}
	if cfg.Timeout != 2*time.Minute {
		t.Fatalf("Timeout = %v", cfg.Timeout)
	}
	if cfg.RequestTimeout != 10*time.Second {
		t.Fatalf("RequestTimeout = %v", cfg.RequestTimeout)
	}
	if cfg.Output != "output.json" {
		t.Fatalf("Output = %q", cfg.Output)
	}
	if cfg.LogFile != "log.txt" {
		t.Fatalf("LogFile = %q", cfg.LogFile)
	}
}

func TestParse_MissingURLs(t *testing.T) {
	_, err := Parse(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "urls") {
		t.Fatalf("error = %v", err)
	}
}

func TestParse_EmptyURLsAfterSplit(t *testing.T) {
	_, err := Parse([]string{"--urls", ", ,,"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParse_NegativeDepth(t *testing.T) {
	_, err := Parse([]string{"--urls", "https://a.com", "--depth", "-1"})
	if err == nil {
		t.Fatal("expected error")
	}
}
