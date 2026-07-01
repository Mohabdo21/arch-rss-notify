package main

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

var pkgDescCache sync.Map

func GetInstalledPackages() (map[string]string, error) {
	cmd := exec.Command("pacman", "-Q")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	packages := make(map[string]string)
	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			packages[fields[0]] = fields[1]
		}
	}

	return packages, nil
}

func GetPackageDescription(name string) (string, error) {
	if val, ok := pkgDescCache.Load(name); ok {
		if desc, ok := val.(string); ok {
			return desc, nil
		}
		pkgDescCache.Delete(name)
	}

	cmd := exec.Command("pacman", "-Qi", name)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to query package %s: %w", name, err)
	}

	desc := parseDescription(string(output))
	pkgDescCache.Store(name, desc)
	return desc, nil
}

func parseDescription(output string) string {
	for line := range strings.SplitSeq(output, "\n") {
		if !strings.HasPrefix(line, "Description") {
			continue
		}
		_, after, ok := strings.Cut(line, ": ")
		if ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}
