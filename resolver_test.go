package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mmcdole/gofeed"
)

func TestStandardResolver_Resolve(t *testing.T) {
	resolver := &StandardResolver{}
	item := &gofeed.Item{Title: "linux 6.1.0-1"}
	pkg, ver, err := resolver.Resolve(context.Background(), item)
	if err != nil || pkg != "linux" || ver != "6.1.0-1" {
		t.Errorf("expected linux 6.1.0-1, got %s %s (err: %v)", pkg, ver, err)
	}
}

func TestAURResolver_Resolve(t *testing.T) {
	// Mock server for AUR RPC
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprint(w, `{"results": [{"Version": "7.1.0-1"}]}`)
		}),
	)
	defer ts.Close()

	resolver := &AURResolver{BaseURL: ts.URL}
	item := &gofeed.Item{Title: "zoom"}
	pkg, ver, err := resolver.Resolve(context.Background(), item)
	if err != nil || pkg != "zoom" || ver != "7.1.0-1" {
		t.Errorf("expected zoom 7.1.0-1, got %s %s (err: %v)", pkg, ver, err)
	}
}
