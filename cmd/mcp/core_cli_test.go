package main

import "testing"

func TestNormalizeCoreCommand_Good(t *testing.T) {
	command, args, err := normalizeCoreCommand("go", []string{"test"})
	if err != nil {
		t.Fatalf("expected command to be allowed: %v", err)
	}
	if command != "go" {
		t.Fatalf("expected go command, got %s", command)
	}
	if len(args) != 1 || args[0] != "test" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestNormalizeCoreCommand_Bad(t *testing.T) {
	if _, _, err := normalizeCoreCommand("rm -rf", nil); err == nil {
		t.Fatalf("expected command to be rejected")
	}
}

func TestNormalizeCoreCommand_Ugly(t *testing.T) {
	command, args, err := normalizeCoreCommand("go test", []string{"-v"})
	if err != nil {
		t.Fatalf("expected command to be allowed: %v", err)
	}
	if command != "go" {
		t.Fatalf("expected go command, got %s", command)
	}
	if len(args) != 2 || args[0] != "test" || args[1] != "-v" {
		t.Fatalf("unexpected args: %#v", args)
	}
}
