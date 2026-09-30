package crawler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"crawler-cli/internal/fetcher"
	"crawler-cli/internal/logger"
)

func newTestLogger(t *testing.T) *logger.Logger {
	t.Helper()
	log, err := logger.New(filepath.Join(t.TempDir(), "test.log"))
	if err != nil {
		t.Fatalf("logger.New: %v", err)
	}
	t.Cleanup(func() { _ = log.Close() })
	return log
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		base, href string
		wantOK     bool
		want       string
	}{
		{"https://example.com/a/", "/b", true, "https://example.com/b"},
		{"https://example.com/a", "c", true, "https://example.com/c"},
		{"https://example.com", "https://example.com/x", true, "https://example.com/x"},
		{"https://example.com", "mailto:a@b.c", false, ""},
		{"https://example.com", "javascript:void(0)", false, ""},
		{"https://example.com", "#", false, ""},
		{"https://example.com", "", false, ""},
	}

	for _, tt := range tests {
		got, ok := resolveURL(tt.base, tt.href)
		if ok != tt.wantOK {
			t.Fatalf("resolveURL(%q, %q) ok=%v, want %v", tt.base, tt.href, ok, tt.wantOK)
		}
		if ok && got != tt.want {
			t.Fatalf("resolveURL(%q, %q) = %q, want %q", tt.base, tt.href, got, tt.want)
		}
	}
}

func TestSameDomain(t *testing.T) {
	if !sameDomain("https://example.com/a", "https://example.com/b") {
		t.Fatal("expected same domain")
	}
	if sameDomain("https://example.com", "https://other.com") {
		t.Fatal("expected different domain")
	}
	if sameDomain("://bad", "https://example.com") {
		t.Fatal("expected false")
	}
}

func TestRun_BuildsTreeAndRespectsDomain(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><title>Home</title><body>
			<a href="/about">About</a>
			<a href="https://evil.example/x">External</a>
		</body></html>`))
	})
	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><title>About</title><body>
			<a href="/">Home</a>
		</body></html>`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(1, fetcher.New(5*time.Second), newTestLogger(t))
	pages, err := c.Run(context.Background(), []string{srv.URL})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("len(pages) = %d", len(pages))
	}
	root := pages[0]
	if root.Title != "Home" {
		t.Fatalf("root.Title = %q", root.Title)
	}
	if len(root.Links) != 1 {
		t.Fatalf("root.Links = %#v", root.Links)
	}
	if root.Links[0].Title != "About" {
		t.Fatalf("child.Title = %q", root.Links[0].Title)
	}
}

func TestRun_NoRepeatVisit(t *testing.T) {
	var hits atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><title>A</title><a href="/b">B</a><a href="/b">B2</a></html>`))
	})
	mux.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><title>B</title><a href="/">A</a></html>`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(2, fetcher.New(5*time.Second), newTestLogger(t))
	_, err := c.Run(context.Background(), []string{srv.URL + "/"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := hits.Load(); got != 2 {
		t.Fatalf("hits = %d, want 2", got)
	}
}

func TestRun_SkipsErrorPages(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><title>Root</title><a href="/gone">Gone</a><a href="/ok">OK</a></html>`))
	})
	mux.HandleFunc("/gone", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><title>OK</title></html>`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(1, fetcher.New(5*time.Second), newTestLogger(t))
	pages, err := c.Run(context.Background(), []string{srv.URL})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	root := pages[0]
	if root.Title != "Root" {
		t.Fatalf("Title = %q", root.Title)
	}
	if len(root.Links) != 1 {
		t.Fatalf("root.Links = %#v", root.Links)
	}
	if root.Links[0].Title != "OK" {
		t.Fatalf("child.Title = %q", root.Links[0].Title)
	}
	if root.Links[0].Resource != srv.URL+"/ok" {
		t.Fatalf("child.Resource = %q", root.Links[0].Resource)
	}
}

func TestRun_ContextCancel(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><title>Slow</title></html>`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-started
		cancel()
	}()

	c := New(1, fetcher.New(5*time.Second), newTestLogger(t))
	pages, err := c.Run(ctx, []string{srv.URL})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run err = %v, want context.Canceled", err)
	}
	if len(pages) != 1 {
		t.Fatalf("len(pages) = %d", len(pages))
	}
	assertNoCrawlerGoroutines(t)
}

func TestRun_TimeoutReturnsDeadlineExceeded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	c := New(1, fetcher.New(5*time.Second), newTestLogger(t))
	pages, err := c.Run(ctx, []string{srv.URL})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run err = %v, want context.DeadlineExceeded", err)
	}
	if len(pages) != 1 {
		t.Fatalf("len(pages) = %d", len(pages))
	}
	assertNoCrawlerGoroutines(t)
}

func TestRun_ThousandsOfLinksStayBounded(t *testing.T) {
	const n = 2000
	var body strings.Builder
	body.WriteString(`<html><title>Root</title>`)
	for i := 0; i < n; i++ {
		fmt.Fprintf(&body, `<a href="/p/%d">p</a>`, i)
	}
	body.WriteString(`</html>`)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(body.String()))
	})
	mux.HandleFunc("/p/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><title>P</title></html>`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	before := runtime.NumGoroutine()
	var peak atomic.Int32
	stop := make(chan struct{})
	var sampleWG sync.WaitGroup
	sampleWG.Add(1)
	go func() {
		defer sampleWG.Done()
		for {
			select {
			case <-stop:
				return
			default:
				g := int32(runtime.NumGoroutine())
				for {
					old := peak.Load()
					if g <= old || peak.CompareAndSwap(old, g) {
						break
					}
				}
				time.Sleep(time.Millisecond)
			}
		}
	}()

	c := New(1, fetcher.New(5*time.Second), newTestLogger(t))
	pages, err := c.Run(context.Background(), []string{srv.URL + "/"})
	close(stop)
	sampleWG.Wait()

	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("len(pages) = %d", len(pages))
	}
	if pages[0].Title != "Root" {
		t.Fatalf("title = %q", pages[0].Title)
	}
	if len(pages[0].Links) != n {
		t.Fatalf("links = %d, want %d", len(pages[0].Links), n)
	}
	if extra := int(peak.Load()) - before; extra > 80 {
		t.Fatalf("peak goroutines above baseline = %d, want <= 80", extra)
	}
	assertNoCrawlerGoroutines(t)
}

func TestRun_CancelWhileQueueIsFull(t *testing.T) {
	const n = 1000
	var body strings.Builder
	body.WriteString(`<html><title>Root</title>`)
	for i := 0; i < n; i++ {
		fmt.Fprintf(&body, `<a href="/p/%d">p</a>`, i)
	}
	body.WriteString(`</html>`)

	blocked := make(chan struct{})
	var started atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(body.String()))
	})
	mux.HandleFunc("/p/", func(w http.ResponseWriter, r *http.Request) {
		if started.Add(1) == int32(maxConcurrent-1) {
			close(blocked)
		}
		<-r.Context().Done()
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := New(1, fetcher.New(5*time.Second), newTestLogger(t))
	errCh := make(chan error, 1)
	go func() {
		_, err := c.Run(ctx, []string{srv.URL + "/"})
		errCh <- err
	}()

	select {
	case <-blocked:
		cancel()
	case <-time.After(5 * time.Second):
		t.Fatal("queue did not fill")
	}

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run err = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run hung while the queue was full")
	}
	assertNoCrawlerGoroutines(t)
}

func crawlerGoroutines() int {
	buf := make([]byte, 64<<10)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return strings.Count(string(buf[:n]), "(*Crawler).Run.func")
		}
		buf = make([]byte, len(buf)*2)
	}
}

func assertNoCrawlerGoroutines(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if crawlerGoroutines() == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("leftover crawler goroutines: %d", crawlerGoroutines())
		}
		time.Sleep(20 * time.Millisecond)
	}
}
