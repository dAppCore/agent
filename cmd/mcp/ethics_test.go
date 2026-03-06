package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEthicsCheck_Good(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("expected repo root: %v", err)
	}

	modalPath := filepath.Join(root, ethicsModalPath)
	modal, err := os.ReadFile(modalPath)
	if err != nil {
		t.Fatalf("expected modal to read: %v", err)
	}
	if len(modal) == 0 {
		t.Fatalf("expected modal content")
	}

	axioms, err := readJSONMap(filepath.Join(root, ethicsAxiomsPath))
	if err != nil {
		t.Fatalf("expected axioms to read: %v", err)
	}
	if len(axioms) == 0 {
		t.Fatalf("expected axioms data")
	}
}

func TestReadJSONMap_Bad(t *testing.T) {
	if _, err := readJSONMap("/missing/file.json"); err == nil {
		t.Fatalf("expected error for missing json")
	}
}
