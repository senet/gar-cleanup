// Package policy loads and evaluates YAML cleanup policies.
package policy

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

// Policy defines the retention rules for a registry repository.
type Policy struct {
	// KeepCount is the minimum number of most-recent images to retain per repo.
	KeepCount int `yaml:"keep"`
	// MaxAge is the maximum age of images to retain (e.g. "30d", "720h").
	MaxAge string `yaml:"max_age"`
	// ProtectTags is a list of regex patterns; matching tags are never deleted.
	ProtectTags []string `yaml:"protect_tags"`
	// DeleteUntagged removes digests with no tags immediately when true.
	DeleteUntagged bool `yaml:"delete_untagged"`
	// DryRun forces dry-run mode regardless of CLI flag (policy-level override).
	DryRun bool `yaml:"dry_run"`

	// compiled regexes, populated by Validate.
	protectRegexps []*regexp.Regexp
	maxAgeDuration time.Duration
}

// Default returns a safe default policy.
func Default() *Policy {
	return &Policy{
		KeepCount:      10,
		MaxAge:         "30d",
		ProtectTags:    []string{`^v\d+\.\d+\.\d+$`},
		DeleteUntagged: true,
		DryRun:         true,
	}
}

// LoadFile reads and validates a YAML policy file. If path is empty the
// Default policy is returned.
func LoadFile(path string) (*Policy, error) {
	if path == "" {
		p := Default()
		if err := p.Validate(); err != nil {
			return nil, err
		}
		return p, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading policy file %q: %w", path, err)
	}

	var p Policy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing policy file %q: %w", path, err)
	}

	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("invalid policy %q: %w", path, err)
	}

	return &p, nil
}

// Validate compiles regexes and parses durations.
func (p *Policy) Validate() error {
	for _, pat := range p.ProtectTags {
		re, err := regexp.Compile(pat)
		if err != nil {
			return fmt.Errorf("invalid protect_tags pattern %q: %w", pat, err)
		}
		p.protectRegexps = append(p.protectRegexps, re)
	}

	if p.MaxAge != "" {
		d, err := parseDuration(p.MaxAge)
		if err != nil {
			return fmt.Errorf("invalid max_age %q: %w", p.MaxAge, err)
		}
		p.maxAgeDuration = d
	}

	return nil
}

// IsProtectedTag returns true if any tag matches a protect_tags pattern.
func (p *Policy) IsProtectedTag(tag string) bool {
	for _, re := range p.protectRegexps {
		if re.MatchString(tag) {
			return true
		}
	}
	return false
}

// MaxAgeDuration returns the parsed max_age as a time.Duration.
func (p *Policy) MaxAgeDuration() time.Duration {
	return p.maxAgeDuration
}

// parseDuration extends time.ParseDuration with day (d) support.
func parseDuration(s string) (time.Duration, error) {
	// handle Nd shorthand (e.g. "30d")
	if len(s) > 1 && s[len(s)-1] == 'd' {
		var days float64
		if _, err := fmt.Sscanf(s[:len(s)-1], "%f", &days); err != nil {
			return 0, fmt.Errorf("cannot parse days from %q", s)
		}
		return time.Duration(days * 24 * float64(time.Hour)), nil
	}
	return time.ParseDuration(s)
}
