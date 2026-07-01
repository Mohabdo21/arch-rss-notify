package main

import (
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(cfg.RSSURLs) == 0 {
		t.Error("Expected RSSURLs to be populated")
	}
	if cfg.CheckInterval == 0 {
		t.Error("Expected CheckInterval to be populated")
	}
	if cfg.StateFile == "" {
		t.Error("Expected StateFile to be populated")
	}
	if cfg.CheckInterval != 10*time.Minute {
		t.Errorf("Expected CheckInterval to be 10m, got %v", cfg.CheckInterval)
	}
	if len(cfg.RSSURLs) != 3 {
		t.Errorf("Expected 3 RSS URLs, got %d", len(cfg.RSSURLs))
	}
	if cfg.FetchRetries != 2 {
		t.Errorf("Expected FetchRetries=2, got %d", cfg.FetchRetries)
	}
	if cfg.FetchBackoff != time.Second {
		t.Errorf("Expected FetchBackoff=1s, got %v", cfg.FetchBackoff)
	}
}
