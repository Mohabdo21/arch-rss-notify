package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"regexp"
	"syscall"
	"time"
)

const version = "0.1.0"

var titleRegex = regexp.MustCompile(`^(\S+)\s+(\S+)`)

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

	items, err := FetchFeeds(ctx, cfg.RSSURLs, fetchCfg)
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

	for _, item := range items {
		matches := titleRegex.FindStringSubmatch(item.Title)
		if len(matches) < 3 {
			continue
		}
		pkg := matches[1]
		version := matches[2]

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
				if err := SendNotification(
					ctx,
					pkg,
					installedVer,
					version,
					desc,
					IsCriticalPackage(pkg),
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

	if err := state.Save(cfg.StateFile); err != nil {
		log.Printf("Error saving state: %v", err)
	}
}
