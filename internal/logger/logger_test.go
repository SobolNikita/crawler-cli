package logger

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogger_StatusAndError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "crawler.log")

	log, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	log.Status("https://example.com", 200)
	log.Error("https://example.com/x", errors.New("boom"))
	log.Error("https://example.com/y", nil)

	if err := log.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "STATUS https://example.com 200") {
		t.Fatalf("missing STATUS line: %q", text)
	}
	if !strings.Contains(text, "ERROR https://example.com/x boom") {
		t.Fatalf("missing ERROR line: %q", text)
	}
	if strings.Count(text, "ERROR") != 1 {
		t.Fatalf("unexpected ERROR count in %q", text)
	}
}

func TestLogger_CloseNil(t *testing.T) {
	var log *Logger
	if err := log.Close(); err != nil {
		t.Fatalf("Close nil: %v", err)
	}
}
