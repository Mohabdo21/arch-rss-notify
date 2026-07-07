package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/mmcdole/gofeed"
)

const version = "0.1.7"

var titleRegex = regexp.MustCompile(`^(\S+)\s+(\S+)`)

type VersionResolver interface {
	Resolve(
		ctx context.Context,
		item *gofeed.Item,
	) (pkg, version string, err error)
}

type StandardResolver struct{}

func (r *StandardResolver) Resolve(
	ctx context.Context,
	item *gofeed.Item,
) (string, string, error) {
	matches := titleRegex.FindStringSubmatch(item.Title)
	if len(matches) < 3 {
		return "", "", fmt.Errorf(
			"title does not match expected pattern: %s",
			item.Title,
		)
	}
	return matches[1], matches[2], nil
}

type AURResolver struct {
	BaseURL string
}

type aurRPCResponse struct {
	Results []struct {
		Version string `json:"Version"`
	} `json:"results"`
}

func (r *AURResolver) Resolve(
	ctx context.Context,
	item *gofeed.Item,
) (string, string, error) {
	// AUR feed items title is usually just the package name
	fields := strings.Fields(item.Title)
	if len(fields) == 0 {
		return "", "", fmt.Errorf("empty title in AUR feed item")
	}
	pkg := fields[0]

	url := fmt.Sprintf("%s/rpc/v5/info/%s", r.BaseURL, pkg)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("AUR RPC returned status %d", resp.StatusCode)
	}

	var res aurRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", "", err
	}

	if len(res.Results) == 0 {
		return "", "", fmt.Errorf("no results found for package %s", pkg)
	}

	return pkg, res.Results[0].Version, nil
}

// consecutiveFetchFailures tracks total-fetch-failure streaks so the ticker

// loop can log escalating warnings without suppressing the next attempt.
var consecutiveFetchFailures int

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	// CLI Overrides
	flagInterval := flag.String("interval", "", "Check interval (e.g. 10m)")
	flagState := flag.String("state", cfg.StateFile, "State file path")
	flag.Parse()

	if *flagInterval != "" {
		if d, err := time.ParseDuration(*flagInterval); err == nil {
			cfg.CheckInterval = d
		} else {
			log.Printf(
				"Warning: invalid interval flag %q, using default %v",
				*flagInterval,
				cfg.CheckInterval,
			)
		}
	}
	cfg.StateFile = *flagState

	state, err := LoadState(cfg.StateFile)
	if err != nil {
		log.Printf("Warning: could not load state, starting fresh: %v", err)
		state, _ = LoadState("non-existent")
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	log.Printf(
		"Starting RSS Notifier v%s (interval: %v)...",
		version,
		cfg.CheckInterval,
	)

	ticker := time.NewTicker(cfg.CheckInterval)
	defer ticker.Stop()

	// Initial check
	checkUpdates(ctx, cfg, state)

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down... saving state")
			if err := state.Save(cfg.StateFile); err != nil {
				log.Printf("Error saving state on shutdown: %v", err)
			}
			return
		case <-ticker.C:
			checkUpdates(ctx, cfg, state)
		}
	}
}

func checkUpdates(ctx context.Context, cfg *Config, state *State) {
	log.Println("Checking for updates...")

	installed, err := GetInstalledPackages()
	if err != nil {
		log.Printf("Error getting installed packages: %v", err)
		return
	}

	fetchCfg := FetchConfig{
		Retries: cfg.FetchRetries,
		Backoff: cfg.FetchBackoff,
	}

	itemsMap, err := FetchFeeds(ctx, cfg.RSSURLs, fetchCfg)
	if err != nil {
		// Total failure - every feed failed even after retries.
		consecutiveFetchFailures++
		log.Printf(
			"Warning: all feeds failed (%d consecutive): %v",
			consecutiveFetchFailures,
			err,
		)
		if consecutiveFetchFailures >= 3 {
			log.Printf(
				"Error: %d consecutive total fetch failures, will retry on next tick",
				consecutiveFetchFailures,
			)
		}
		return // do not rewrite state on total failure
	}
	consecutiveFetchFailures = 0

	resolvers := make(map[string]VersionResolver)
	for _, url := range cfg.RSSURLs {
		if strings.Contains(url, "aur.archlinux.org/rss/modified") {
			resolvers[url] = &AURResolver{BaseURL: "https://aur.archlinux.org"}
		} else {
			resolvers[url] = &StandardResolver{}
		}
	}

	for url, items := range itemsMap {
		resolver, ok := resolvers[url]
		if !ok {
			log.Printf("Warning: no resolver found for feed %s", url)
			continue
		}

		for _, item := range items {
			pkg, version, err := resolver.Resolve(ctx, item)
			if err != nil {
				log.Printf(
					"Warning: could not resolve version for item in %s: %v",
					url,
					err,
				)
				continue
			}

			if installedVer, ok := installed[pkg]; ok {
				if installedVer != version &&
					state.ShouldNotify(pkg, version) {
					log.Printf(
						"Update found for %s: %s -> %s",
						pkg,
						installedVer,
						version,
					)
					desc, err := GetPackageDescription(pkg)
					if err != nil {
						log.Printf(
							"Warning: could not get description for %s: %v",
							pkg,
							err,
						)
					}
					isAUR := strings.Contains(url, "aur.archlinux.org")
					if err := SendNotification(
						ctx,
						pkg,
						installedVer,
						version,
						desc,
						IsCriticalPackage(pkg),
						isAUR,
					); err != nil {
						log.Printf(
							"Error sending notification for %s: %v",
							pkg,
							err,
						)
					}
					state.MarkNotified(pkg, version)
				}
			}
		}
	}

	if err := state.Save(cfg.StateFile); err != nil {
		log.Printf("Error saving state: %v", err)
	}
}
