package main

import (
	"encoding/json"
	"os"
)

type Settings struct {
	Vars      map[string]string   `json:"vars"`
	Commands  map[string][]string `json:"commands"`
	OutputDir string              `json:"output_dir"`
	InputDir  string              `json:"input_dir"`
}

// NewSettingsFromJSON creates a new Settings instance from a JSON string.
func NewSettingsFromJSON(path string) (s Settings, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	return s, json.Unmarshal(b, &s)
}
