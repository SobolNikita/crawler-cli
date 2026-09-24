package config

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

type Config struct {
	URLs           []string
	Depth          int
	Timeout        time.Duration
	RequestTimeout time.Duration
	Output         string
	LogFile        string
}

func Parse(args []string) (*Config, error) {
	fs := flag.NewFlagSet("crawler-cli", flag.ContinueOnError)

	urls := fs.String("urls", "", "URLs to crawl, comma-separated")
	depth := fs.Int("depth", 0, "Maximum depth to crawl")
	timeout := fs.Duration("timeout", 2*time.Minute, "Overall timeout")
	requestTimeout := fs.Duration("request-timeout", 10*time.Second, "Timeout for a single request")
	output := fs.String("output", "output.json", "Path to JSON output file")
	logFile := fs.String("log", "log.txt", "Path to log file")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if *urls == "" {
		return nil, fmt.Errorf("urls is required")
	}
	if *depth < 0 {
		return nil, fmt.Errorf("depth must be >= 0")
	}

	splitUrls := strings.Split(*urls, ",")
	resultUrls := make([]string, 0, len(splitUrls))
	for _, url := range splitUrls {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}
		resultUrls = append(resultUrls, url)
	}

	if len(resultUrls) == 0 {
		return nil, fmt.Errorf("urls is required")
	}

	return &Config{
		URLs:           resultUrls,
		Depth:          *depth,
		Timeout:        *timeout,
		RequestTimeout: *requestTimeout,
		Output:         *output,
		LogFile:        *logFile,
	}, nil
}
