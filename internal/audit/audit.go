// Package audit writes structured deletion events to BigQuery for
// cost-tracking and compliance dashboards.
package audit

import (
	"context"
	"log/slog"
	"time"
)

// Event is a single deletion or dry-run candidate event.
type Event struct {
	Timestamp    time.Time `json:"timestamp"`
	Registry     string    `json:"registry"`
	Repository   string    `json:"repository"`
	Digest       string    `json:"digest"`
	Tags         []string  `json:"tags"`
	SizeBytes    int64     `json:"size_bytes"`
	Reason       string    `json:"reason"`
	DryRun       bool      `json:"dry_run"`
	K8sProtected bool      `json:"k8s_protected"`
}

// Writer persists audit events.
type Writer interface {
	Write(ctx context.Context, event Event) error
	Close() error
}

// LogWriter is a Writer that emits events as structured log lines.
// It is the default when BigQuery is not configured, and is also used in
// dry-run mode so operators can review the full deletion plan.
type LogWriter struct{}

func (LogWriter) Write(ctx context.Context, e Event) error {
	slog.InfoContext(ctx, "audit event",
		"timestamp", e.Timestamp,
		"registry", e.Registry,
		"repository", e.Repository,
		"digest", e.Digest,
		"tags", e.Tags,
		"size_bytes", e.SizeBytes,
		"reason", e.Reason,
		"dry_run", e.DryRun,
		"k8s_protected", e.K8sProtected,
	)
	return nil
}

func (LogWriter) Close() error { return nil }

// Compile-time interface check.
var _ Writer = LogWriter{}

// BigQueryWriter streams events to a BigQuery table.
// Table schema mirrors the Event struct fields.
//
// TODO(phase4): implement using cloud.google.com/go/bigquery.
type BigQueryWriter struct {
	ProjectID string
	DatasetID string
	TableID   string
}

func (b *BigQueryWriter) Write(ctx context.Context, e Event) error {
	// TODO(phase4): stream insert via bigquery.Client.
	slog.InfoContext(ctx, "[bigquery stub] would write audit event",
		"project", b.ProjectID,
		"dataset", b.DatasetID,
		"table", b.TableID,
		"digest", e.Digest,
	)
	return nil
}

func (b *BigQueryWriter) Close() error { return nil }

var _ Writer = (*BigQueryWriter)(nil)
