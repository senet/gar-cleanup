// Package engine implements the core image cleanup logic.
package engine

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/senet/gar-cleanup/internal/audit"
	"github.com/senet/gar-cleanup/internal/k8s"
	"github.com/senet/gar-cleanup/internal/policy"
	"github.com/senet/gar-cleanup/internal/registry"
)

// Config holds engine runtime configuration.
type Config struct {
	Project    string
	Location   string
	DryRun     bool
	PolicyFile string
}

// Engine orchestrates the cleanup run: inventory → policy evaluation →
// K8s guard → delete/audit.
type Engine struct {
	cfg      Config
	policy   *policy.Policy
	registry registry.Client
	guard    k8s.Guard
	auditor  audit.Writer
}

// New creates an Engine. In production the registry client is a GARClient;
// in tests a FakeClient can be injected via the options below.
func New(cfg Config, pol *policy.Policy, opts ...Option) (*Engine, error) {
	e := &Engine{
		cfg:     cfg,
		policy:  pol,
		guard:   k8s.NoopGuard{},
		auditor: audit.LogWriter{},
	}

	for _, o := range opts {
		o(e)
	}

	if e.registry == nil {
		client, err := registry.NewGARClient(cfg.Project, cfg.Location)
		if err != nil {
			return nil, fmt.Errorf("creating GAR client: %w", err)
		}
		e.registry = client
	}

	return e, nil
}

// Option is a functional option for Engine.
type Option func(*Engine)

// WithRegistry injects a custom registry client (used in tests).
func WithRegistry(c registry.Client) Option {
	return func(e *Engine) { e.registry = c }
}

// WithGuard injects a custom K8s guard (used in tests).
func WithGuard(g k8s.Guard) Option {
	return func(e *Engine) { e.guard = g }
}

// WithAuditor injects a custom audit writer (used in tests).
func WithAuditor(a audit.Writer) Option {
	return func(e *Engine) { e.auditor = a }
}

// Run executes a full cleanup cycle.
func (e *Engine) Run(ctx context.Context) error {
	mode := "dry-run"
	if !e.cfg.DryRun {
		mode = "live"
	}
	slog.InfoContext(ctx, "starting cleanup run",
		"project", e.cfg.Project,
		"location", e.cfg.Location,
		"mode", mode,
		"policy_file", e.cfg.PolicyFile,
	)

	// Phase 1: build K8s protected set.
	protected, err := e.guard.ProtectedDigests(ctx)
	if err != nil {
		return fmt.Errorf("building K8s protected set: %w", err)
	}
	slog.InfoContext(ctx, "K8s protected digests", "count", len(protected))

	// Phase 2: list repositories.
	parent := fmt.Sprintf("projects/%s/locations/%s", e.cfg.Project, e.cfg.Location)
	repos, err := e.registry.ListRepositories(ctx, parent)
	if err != nil {
		return fmt.Errorf("listing repositories: %w", err)
	}
	slog.InfoContext(ctx, "found repositories", "count", len(repos))

	var totalDeleted, totalProtected, totalSkipped int

	for _, repo := range repos {
		deleted, prot, skipped, err := e.processRepo(ctx, repo, protected)
		if err != nil {
			slog.ErrorContext(ctx, "error processing repository", "repo", repo, "error", err)
			continue
		}
		totalDeleted += deleted
		totalProtected += prot
		totalSkipped += skipped
	}

	slog.InfoContext(ctx, "cleanup run complete",
		"mode", mode,
		"deleted", totalDeleted,
		"k8s_protected", totalProtected,
		"retained", totalSkipped,
	)
	return nil
}

// processRepo applies policy to a single repository and returns the number
// of images deleted, K8s-protected, and retained.
func (e *Engine) processRepo(ctx context.Context, repo string, protected map[string]bool) (deleted, k8sProtected, retained int, err error) {
	images, err := e.registry.ListImages(ctx, repo)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("listing images in %q: %w", repo, err)
	}

	// Sort by PushedAt descending so we keep the N most-recent first.
	sortByPushedAtDesc(images)

	cutoff := time.Now().Add(-e.policy.MaxAgeDuration())

	for i, img := range images {
		reason, action := e.evaluateImage(ctx, img, i, cutoff, protected)
		_ = reason

		switch action {
		case actionDelete:
			if err2 := e.deleteImage(ctx, repo, img, reason); err2 != nil {
				slog.ErrorContext(ctx, "failed to delete image", "digest", img.Digest, "error", err2)
			}
			deleted++
		case actionK8sProtect:
			slog.InfoContext(ctx, "image protected by K8s guard",
				"digest", img.Digest, "tags", img.Tags)
			k8sProtected++
			if err2 := e.auditor.Write(ctx, audit.Event{
				Timestamp:    time.Now(),
				Registry:     "gar",
				Repository:   repo,
				Digest:       img.Digest,
				Tags:         img.Tags,
				SizeBytes:    img.SizeBytes,
				Reason:       reason,
				DryRun:       e.cfg.DryRun,
				K8sProtected: true,
			}); err2 != nil {
				slog.WarnContext(ctx, "audit write failed", "error", err2)
			}
		default:
			retained++
		}
	}

	return deleted, k8sProtected, retained, nil
}

type imageAction int

const (
	actionRetain    imageAction = iota
	actionDelete
	actionK8sProtect
)

func (e *Engine) evaluateImage(ctx context.Context, img registry.Image, rankIdx int, cutoff time.Time, protected map[string]bool) (reason string, action imageAction) {
	// K8s guard takes precedence.
	if protected[img.Digest] {
		return "k8s_in_use", actionK8sProtect
	}

	// Untagged images are deleted immediately if policy says so.
	if len(img.Tags) == 0 && e.policy.DeleteUntagged {
		return "untagged", actionDelete
	}

	// Any semver-protected tag means retain.
	for _, tag := range img.Tags {
		if e.policy.IsProtectedTag(tag) {
			return "protected_tag", actionRetain
		}
	}

	// Keep the N most-recent images.
	if rankIdx < e.policy.KeepCount {
		return "within_keep_count", actionRetain
	}

	// Delete if older than max_age.
	if e.policy.MaxAgeDuration() > 0 && img.PushedAt.Before(cutoff) {
		return "exceeds_max_age", actionDelete
	}

	return "retained", actionRetain
}

func (e *Engine) deleteImage(ctx context.Context, repo string, img registry.Image, reason string) error {
	// Untag first.
	for _, tag := range img.Tags {
		if err := e.registry.UntagImage(ctx, repo, img.Digest, tag, e.cfg.DryRun); err != nil {
			return fmt.Errorf("untagging %s: %w", tag, err)
		}
	}

	// Then delete digest.
	if err := e.registry.DeleteImage(ctx, repo, img.Digest, e.cfg.DryRun); err != nil {
		return fmt.Errorf("deleting digest %s: %w", img.Digest, err)
	}

	if err := e.auditor.Write(ctx, audit.Event{
		Timestamp:  time.Now(),
		Registry:   "gar",
		Repository: repo,
		Digest:     img.Digest,
		Tags:       img.Tags,
		SizeBytes:  img.SizeBytes,
		Reason:     reason,
		DryRun:     e.cfg.DryRun,
	}); err != nil {
		slog.WarnContext(ctx, "audit write failed", "error", err)
	}

	return nil
}

// sortByPushedAtDesc sorts images newest-first in-place.
func sortByPushedAtDesc(images []registry.Image) {
	for i := 1; i < len(images); i++ {
		for j := i; j > 0 && images[j].PushedAt.After(images[j-1].PushedAt); j-- {
			images[j], images[j-1] = images[j-1], images[j]
		}
	}
}
