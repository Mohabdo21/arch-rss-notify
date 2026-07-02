package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type State struct {
	mu       sync.RWMutex
	Notified map[string]string `json:"notified"`
	dirty    bool
}

func LoadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{Notified: make(map[string]string)}, nil
		}
		return nil, fmt.Errorf("could not read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return &State{Notified: make(map[string]string)}, nil
	}

	if state.Notified == nil {
		state.Notified = make(map[string]string)
	}

	return &state, nil
}

func (s *State) Save(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.dirty {
		return nil
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal state: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create state directory: %w", err)
	}
	s.dirty = false
	return os.WriteFile(path, data, 0644)
}

func (s *State) ShouldNotify(pkg, version string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lastVersion, ok := s.Notified[pkg]
	if !ok || lastVersion != version {
		return true
	}
	return false
}

func (s *State) MarkNotified(pkg, version string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Notified[pkg] = version
	s.dirty = true
}
