package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var criticalPrefixes = []string{
	"linux",
	"nvidia",
	"mesa",
	"v4l2loopback",
	"dkms",
	"sbctl",
}

var criticalExact = map[string]bool{
	"glibc":        true,
	"gcc":          true,
	"gcc-libs":     true,
	"gcc-fortran":  true,
	"systemd":      true,
	"systemd-libs": true,
	"dbus":         true,
	"openssl":      true,
	"gnutls":       true,
	"pam":          true,
}

func IsCriticalPackage(pkg string) bool {
	if criticalExact[pkg] {
		return true
	}
	for _, prefix := range criticalPrefixes {
		if strings.HasPrefix(pkg, prefix) {
			return true
		}
	}
	return false
}

func SendNotification(
	ctx context.Context,
	pkg, oldVersion, version, description string,
	critical bool,
) error {
	icon := "software-update-available"
	urgency := "normal"
	summary := "Package Update Available"
	if critical {
		icon = "dialog-warning"
		urgency = "critical"
		summary = "Critical Update - Reboot May Be Required"
	}
	body := fmt.Sprintf("%s updated from %s to %s", pkg, oldVersion, version)
	if description != "" {
		body += "\n" + description
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx,
		"notify-send",
		"--app-name=Arch Update Notifier",
		"--icon="+icon,
		"--urgency="+urgency,
		"--category=updates",
		summary,
		body,
	)
	return cmd.Run()
}
