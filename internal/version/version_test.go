package version

import "testing"

func TestVersionVariablesNotEmpty(t *testing.T) {
	if Version == "" {
		t.Fatal("Version should not be empty")
	}
	if Commit == "" {
		t.Fatal("Commit should not be empty")
	}
	if BuildDate == "" {
		t.Fatal("BuildDate should not be empty")
	}
}

func TestVersionVariablesOverride(t *testing.T) {
	origVersion := Version
	origCommit := Commit
	origBuildDate := BuildDate
	defer func() {
		Version = origVersion
		Commit = origCommit
		BuildDate = origBuildDate
	}()

	Version = "1.2.3"
	Commit = "abcdef"
	BuildDate = "2025-01-01T00:00:00Z"

	if Version != "1.2.3" {
		t.Fatalf("Version = %s, want 1.2.3", Version)
	}
	if Commit != "abcdef" {
		t.Fatalf("Commit = %s, want abcdef", Commit)
	}
	if BuildDate != "2025-01-01T00:00:00Z" {
		t.Fatalf("BuildDate = %s, want 2025-01-01T00:00:00Z", BuildDate)
	}
}
