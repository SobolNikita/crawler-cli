package crawler

import (
	"context"
	"net/url"
	"strings"
	"sync"

	"crawler-cli/internal/fetcher"
	"crawler-cli/internal/logger"
	"crawler-cli/internal/model"
	"crawler-cli/internal/parser"
)

const maxConcurrent = 10

type job struct {
	rawURL string
	depth  int
	root   string
}

type Crawler struct {
	fetcher  *fetcher.Fetcher
	logger   *logger.Logger
	maxDepth int
}

func New(maxDepth int, f *fetcher.Fetcher, log *logger.Logger) *Crawler {
	return &Crawler{
		fetcher:  f,
		logger:   log,
		maxDepth: maxDepth,
	}
}

func (c *Crawler) Run(ctx context.Context, startURLs []string) ([]model.Page, error) {
	var mu sync.Mutex
	visited := make(map[string]struct{})
	pages := make(map[string]*model.Page)
	children := make(map[string][]string)

	jobs := make(chan job, 1024)
	var wg sync.WaitGroup

	enqueue := func(j job) bool {
		mu.Lock()
		if _, seen := visited[j.rawURL]; seen {
			mu.Unlock()
			return false
		}
		visited[j.rawURL] = struct{}{}
		mu.Unlock()

		wg.Add(1)
		select {
		case <-ctx.Done():
			wg.Done()
			return false
		case jobs <- j:
			return true
		}
	}

	var workersDone sync.WaitGroup
	workersDone.Add(maxConcurrent)
	for range maxConcurrent {
		go func() {
			defer workersDone.Done()
			for j := range jobs {
				c.process(ctx, j, &mu, pages, children, enqueue)
				wg.Done()
			}
		}()
	}

	for _, u := range startURLs {
		enqueue(job{rawURL: u, depth: 0, root: u})
	}

	go func() {
		wg.Wait()
		close(jobs)
	}()

	workersDone.Wait()

	var build func(string) model.Page
	build = func(u string) model.Page {
		links := make([]model.Page, 0)
		for _, child := range children[u] {
			links = append(links, build(child))
		}

		mu.Lock()
		p := pages[u]
		mu.Unlock()

		if p == nil {
			return model.Page{Resource: u, Links: links}
		}
		return model.Page{Resource: p.Resource, Title: p.Title, Links: links}
	}

	result := make([]model.Page, 0, len(startURLs))
	for _, u := range startURLs {
		result = append(result, build(u))
	}
	return result, nil
}

func (c *Crawler) process(
	ctx context.Context,
	j job,
	mu *sync.Mutex,
	pages map[string]*model.Page,
	children map[string][]string,
	enqueue func(job) bool,
) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	res := c.fetcher.Fetch(ctx, j.rawURL)

	if res.StatusCode != 0 {
		c.logger.Status(j.rawURL, res.StatusCode)
	}
	if res.Err != nil {
		c.logger.Error(j.rawURL, res.Err)
		return
	}

	parsed, err := parser.ParseHTML(res.Body)
	if err != nil {
		c.logger.Error(j.rawURL, err)
		return
	}

	mu.Lock()
	pages[j.rawURL] = &model.Page{
		Resource: j.rawURL,
		Title:    parsed.Title,
	}
	mu.Unlock()

	if j.depth >= c.maxDepth {
		return
	}

	for _, href := range parsed.Links {
		abs, ok := resolveURL(j.rawURL, href)
		if !ok {
			continue
		}
		if !sameDomain(j.root, abs) {
			continue
		}

		child := job{rawURL: abs, depth: j.depth + 1, root: j.root}
		if enqueue(child) {
			mu.Lock()
			children[j.rawURL] = append(children[j.rawURL], abs)
			mu.Unlock()
		}
	}
}

func resolveURL(base, href string) (string, bool) {
	href = strings.TrimSpace(href)
	if href == "" || href == "#" {
		return "", false
	}

	b, err := url.Parse(base)
	if err != nil {
		return "", false
	}
	ref, err := url.Parse(href)
	if err != nil {
		return "", false
	}

	switch strings.ToLower(ref.Scheme) {
	case "mailto", "javascript", "tel", "data":
		return "", false
	}

	abs := b.ResolveReference(ref)
	if abs.Scheme != "http" && abs.Scheme != "https" {
		return "", false
	}
	return abs.String(), true
}

func sameDomain(base, link string) bool {
	u1, err1 := url.Parse(base)
	u2, err2 := url.Parse(link)
	if err1 != nil || err2 != nil {
		return false
	}
	return u1.Hostname() == u2.Hostname()
}
