package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	RSSURLs       []string
	CheckInterval time.Duration
	StateFile     string
	FetchRetries  int
	FetchBackoff  time.Duration
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	rssURLs := defaultRSSURLs()
	if env := os.Getenv("RSS_URL"); env != "" {
		rssURLs = strings.Split(env, ",")
		for i, u := range rssURLs {
			rssURLs[i] = strings.TrimSpace(u)
		}
	}

	checkIntervalStr := os.Getenv("CHECK_INTERVAL")
	var checkInterval time.Duration
	var err error
	if checkIntervalStr != "" {
		checkInterval, err = time.ParseDuration(checkIntervalStr)
		if err != nil {
			return nil, fmt.Errorf("invalid CHECK_INTERVAL: %w", err)
		}
	} else {
		checkInterval = 10 * time.Minute
	}

	stateFile := os.Getenv("STATE_FILE")
	if stateFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			stateFile = "state.json"
		} else {
			stateFile = filepath.Join(
				home,
				".local",
				"share",
				"rss-notifier",
				"state.json",
			)
		}
	}

	fetchRetries := 2
	if env := os.Getenv("FETCH_RETRIES"); env != "" {
		if v, err := strconv.Atoi(env); err == nil && v >= 0 {
			fetchRetries = v
		}
	}

	fetchBackoff := 1 * time.Second
	if env := os.Getenv("FETCH_BACKOFF"); env != "" {
		if d, err := time.ParseDuration(env); err == nil && d > 0 {
			fetchBackoff = d
		}
	}

	return &Config{
		RSSURLs:       rssURLs,
		CheckInterval: checkInterval,
		StateFile:     stateFile,
		FetchRetries:  fetchRetries,
		FetchBackoff:  fetchBackoff,
	}, nil
}

func defaultRSSURLs() []string {
	return []string{
		"https://archlinux.org/feeds/packages/x86_64/core/",
		"https://archlinux.org/feeds/packages/x86_64/extra/",
		"https://archlinux.org/feeds/packages/x86_64/multilib/",
		"https://aur.archlinux.org/rss/modified",
	}
}
