package fetcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetch_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><title>OK</title></html>"))
	}))
	defer srv.Close()

	f := New(5 * time.Second)
	res := f.Fetch(context.Background(), srv.URL)
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("StatusCode = %d", res.StatusCode)
	}
	if !strings.Contains(string(res.Body), "OK") {
		t.Fatalf("Body = %q", res.Body)
	}
}

func TestFetch_NotHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"a":1}`))
	}))
	defer srv.Close()

	f := New(5 * time.Second)
	res := f.Fetch(context.Background(), srv.URL)
	if res.Err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(res.Err.Error(), "not html") {
		t.Fatalf("error = %v", res.Err)
	}
}

func TestFetch_BadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "text/html")
	}))
	defer srv.Close()

	f := New(5 * time.Second)
	res := f.Fetch(context.Background(), srv.URL)
	if res.Err == nil {
		t.Fatal("expected error")
	}
	if res.StatusCode != 404 {
		t.Fatalf("StatusCode = %d", res.StatusCode)
	}
}

func TestFetch_SkipRedirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("ok"))
	}))
	defer target.Close()

	redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redir.Close()

	f := New(5 * time.Second)
	res := f.Fetch(context.Background(), redir.URL)
	if res.Err == nil {
		t.Fatal("expected error")
	}
	if res.StatusCode != http.StatusFound {
		t.Fatalf("StatusCode = %d", res.StatusCode)
	}
}

func TestFetch_CanceledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("late"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	f := New(5 * time.Second)
	res := f.Fetch(ctx, srv.URL)
	if res.Err == nil {
		t.Fatal("expected error")
	}
}
