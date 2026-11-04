package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestChooseDataPathUsesConfigWhenLocalDatabaseIsMissing(t *testing.T) {
	root := t.TempDir()
	localData := filepath.Join(root, "data", "xiangying-notes.sqlite")
	if err := os.MkdirAll(filepath.Dir(localData), 0755); err != nil {
		t.Fatal(err)
	}
	configDir := filepath.Join(root, "config")

	got := chooseDataPath(localData, configDir, nil)
	want := filepath.Join(configDir, "Xiangying", "xiangying-notes.sqlite")
	if got != want {
		t.Fatalf("chooseDataPath() = %q, want %q", got, want)
	}
}

func TestChooseDataPathPrefersExistingLocalDatabase(t *testing.T) {
	root := t.TempDir()
	localData := filepath.Join(root, "data", "xiangying-notes.sqlite")
	if err := os.MkdirAll(filepath.Dir(localData), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localData, []byte("existing database"), 0600); err != nil {
		t.Fatal(err)
	}

	got := chooseDataPath(localData, filepath.Join(root, "config"), nil)
	if got != localData {
		t.Fatalf("chooseDataPath() = %q, want existing local path %q", got, localData)
	}
}

func TestChooseDataPathFallsBackLocallyWithoutConfigDirectory(t *testing.T) {
	localData := filepath.Join(t.TempDir(), "data", "xiangying-notes.sqlite")
	got := chooseDataPath(localData, "", errors.New("config directory unavailable"))
	if got != localData {
		t.Fatalf("chooseDataPath() = %q, want fallback path %q", got, localData)
	}
}
