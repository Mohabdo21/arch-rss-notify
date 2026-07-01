package main

import (
	"testing"
)

func TestGetInstalledPackages(t *testing.T) {
	pkgs, err := GetInstalledPackages()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(pkgs) == 0 {
		t.Error("Expected at least one installed package")
	}
}
