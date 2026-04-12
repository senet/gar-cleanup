package policy_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/senet/gar-cleanup/internal/policy"
)

func TestDefault(t *testing.T) {
	p := policy.Default()
	if err := p.Validate(); err != nil {
		t.Fatalf("default policy invalid: %v", err)
	}
	if p.KeepCount != 10 {
		t.Errorf("expected keep=10, got %d", p.KeepCount)
	}
	if !p.DryRun {
		t.Error("expected dry_run=true by default")
	}
}

func TestIsProtectedTag(t *testing.T) {
	p := policy.Default()
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		tag       string
		protected bool
	}{
		{"v1.2.3", true},
		{"v0.0.1", true},
		{"latest", false},
		{"main", false},
		{"sha-abc123", false},
	}

	for _, tc := range tests {
		got := p.IsProtectedTag(tc.tag)
		if got != tc.protected {
			t.Errorf("IsProtectedTag(%q) = %v, want %v", tc.tag, got, tc.protected)
		}
	}
}

func TestMaxAgeDuration(t *testing.T) {
	p := &policy.Policy{
		KeepCount: 5,
		MaxAge:    "30d",
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	want := 30 * 24 * time.Hour
	if p.MaxAgeDuration() != want {
		t.Errorf("MaxAgeDuration() = %v, want %v", p.MaxAgeDuration(), want)
	}
}

func TestLoadFile(t *testing.T) {
	content := `
keep: 5
max_age: 14d
protect_tags:
  - "^v\\d+\\.\\d+\\.\\d+$"
  - "^stable$"
delete_untagged: true
dry_run: true
`
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	p, err := policy.LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if p.KeepCount != 5 {
		t.Errorf("keep = %d, want 5", p.KeepCount)
	}
	if !p.IsProtectedTag("v1.0.0") {
		t.Error("v1.0.0 should be protected")
	}
	if !p.IsProtectedTag("stable") {
		t.Error("stable should be protected")
	}
	if p.IsProtectedTag("latest") {
		t.Error("latest should not be protected")
	}
}

func TestParseDurationHours(t *testing.T) {
	p := &policy.Policy{
		KeepCount: 5,
		MaxAge:    "72h",
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	want := 72 * time.Hour
	if p.MaxAgeDuration() != want {
		t.Errorf("MaxAgeDuration() = %v, want %v", p.MaxAgeDuration(), want)
	}
}
