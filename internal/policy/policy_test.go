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

func TestLoadFile_EmptyPath_ReturnsDefault(t *testing.T) {
	p, err := policy.LoadFile("")
	if err != nil {
		t.Fatalf("LoadFile(\"\") returned error: %v", err)
	}
	if p.KeepCount != 10 {
		t.Errorf("expected default keep=10, got %d", p.KeepCount)
	}
	if !p.DryRun {
		t.Error("expected default dry_run=true")
	}
}

func TestLoadFile_NonexistentFile(t *testing.T) {
	_, err := policy.LoadFile("/nonexistent/path/policy.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoadFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("{{{{not yaml"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := policy.LoadFile(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadFile_InvalidRegex(t *testing.T) {
	content := `
keep: 5
max_age: 30d
protect_tags:
  - "[invalid"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := policy.LoadFile(path)
	if err == nil {
		t.Fatal("expected error for invalid regex in protect_tags")
	}
}

func TestLoadFile_InvalidMaxAge(t *testing.T) {
	content := `
keep: 5
max_age: "notaduration"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := policy.LoadFile(path)
	if err == nil {
		t.Fatal("expected error for invalid max_age")
	}
}

func TestValidate_EmptyMaxAge(t *testing.T) {
	p := &policy.Policy{
		KeepCount: 5,
		MaxAge:    "",
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() returned error for empty max_age: %v", err)
	}
	if p.MaxAgeDuration() != 0 {
		t.Errorf("expected zero duration for empty max_age, got %v", p.MaxAgeDuration())
	}
}

func TestValidate_NoProtectTags(t *testing.T) {
	p := &policy.Policy{
		KeepCount:   5,
		MaxAge:      "10d",
		ProtectTags: nil,
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() returned error for nil protect_tags: %v", err)
	}
}

func TestParseDuration_FractionalDays(t *testing.T) {
	p := &policy.Policy{
		KeepCount: 1,
		MaxAge:    "1.5d",
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}
	want := time.Duration(1.5 * 24 * float64(time.Hour))
	if p.MaxAgeDuration() != want {
		t.Errorf("MaxAgeDuration() = %v, want %v", p.MaxAgeDuration(), want)
	}
}

func TestParseDuration_LargeDays(t *testing.T) {
	p := &policy.Policy{
		KeepCount: 1,
		MaxAge:    "365d",
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}
	want := 365 * 24 * time.Hour
	if p.MaxAgeDuration() != want {
		t.Errorf("MaxAgeDuration() = %v, want %v", p.MaxAgeDuration(), want)
	}
}

func TestParseDuration_Minutes(t *testing.T) {
	p := &policy.Policy{
		KeepCount: 1,
		MaxAge:    "90m",
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}
	want := 90 * time.Minute
	if p.MaxAgeDuration() != want {
		t.Errorf("MaxAgeDuration() = %v, want %v", p.MaxAgeDuration(), want)
	}
}

func TestParseDuration_InvalidDaySuffix(t *testing.T) {
	p := &policy.Policy{
		KeepCount: 1,
		MaxAge:    "abcd",
	}
	err := p.Validate()
	if err == nil {
		t.Fatal("expected error for invalid day suffix 'abcd'")
	}
}

func TestIsProtectedTag_EmptyPatterns(t *testing.T) {
	p := &policy.Policy{
		KeepCount:   5,
		ProtectTags: nil,
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if p.IsProtectedTag("v1.0.0") {
		t.Error("expected no tag to be protected with empty patterns")
	}
}

func TestIsProtectedTag_MultiplePatterns(t *testing.T) {
	p := &policy.Policy{
		KeepCount:   5,
		MaxAge:      "30d",
		ProtectTags: []string{`^v\d+\.\d+\.\d+$`, `^release-`, `^stable$`},
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		tag       string
		protected bool
	}{
		{"v1.2.3", true},
		{"release-2024-01", true},
		{"stable", true},
		{"latest", false},
		{"dev", false},
		{"release", false}, // no dash after release
	}

	for _, tc := range tests {
		got := p.IsProtectedTag(tc.tag)
		if got != tc.protected {
			t.Errorf("IsProtectedTag(%q) = %v, want %v", tc.tag, got, tc.protected)
		}
	}
}

func TestDefault_Values(t *testing.T) {
	p := policy.Default()
	if p.MaxAge != "30d" {
		t.Errorf("expected max_age=30d, got %s", p.MaxAge)
	}
	if !p.DeleteUntagged {
		t.Error("expected delete_untagged=true")
	}
	if len(p.ProtectTags) != 1 {
		t.Errorf("expected 1 protect_tags pattern, got %d", len(p.ProtectTags))
	}
}

func TestValidate_Idempotent(t *testing.T) {
	p := &policy.Policy{
		KeepCount:   5,
		MaxAge:      "30d",
		ProtectTags: []string{`^v\d+$`},
	}
	// Validate twice - should not accumulate regexes erroneously
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	// Tags should still match correctly after double validation
	if !p.IsProtectedTag("v1") {
		t.Error("v1 should be protected after double Validate")
	}
}
