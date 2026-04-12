package audit_test

import (
	"context"
	"testing"
	"time"

	"github.com/senet/gar-cleanup/internal/audit"
)

func TestLogWriter_Write(t *testing.T) {
	w := audit.LogWriter{}
	err := w.Write(context.Background(), audit.Event{
		Timestamp:    time.Now(),
		Registry:     "gar",
		Repository:   "us-central1-docker.pkg.dev/proj/repo/app",
		Digest:       "sha256:abc123",
		Tags:         []string{"v1.0.0", "latest"},
		SizeBytes:    1024,
		Reason:       "exceeds_max_age",
		DryRun:       true,
		K8sProtected: false,
	})
	if err != nil {
		t.Errorf("LogWriter.Write() returned error: %v", err)
	}
}

func TestLogWriter_WriteMinimalEvent(t *testing.T) {
	w := audit.LogWriter{}
	err := w.Write(context.Background(), audit.Event{})
	if err != nil {
		t.Errorf("LogWriter.Write() with zero-value event returned error: %v", err)
	}
}

func TestLogWriter_Close(t *testing.T) {
	w := audit.LogWriter{}
	if err := w.Close(); err != nil {
		t.Errorf("LogWriter.Close() returned error: %v", err)
	}
}

func TestBigQueryWriter_Write(t *testing.T) {
	w := &audit.BigQueryWriter{
		ProjectID: "test-project",
		DatasetID: "test-dataset",
		TableID:   "test-table",
	}
	err := w.Write(context.Background(), audit.Event{
		Timestamp:  time.Now(),
		Registry:   "gar",
		Repository: "us-central1-docker.pkg.dev/proj/repo/app",
		Digest:     "sha256:abc123",
		Tags:       []string{"v1.0.0"},
		SizeBytes:  2048,
		Reason:     "untagged",
		DryRun:     false,
	})
	if err != nil {
		t.Errorf("BigQueryWriter.Write() returned error: %v", err)
	}
}

func TestBigQueryWriter_Close(t *testing.T) {
	w := &audit.BigQueryWriter{
		ProjectID: "test-project",
		DatasetID: "test-dataset",
		TableID:   "test-table",
	}
	if err := w.Close(); err != nil {
		t.Errorf("BigQueryWriter.Close() returned error: %v", err)
	}
}

func TestLogWriter_ImplementsWriterInterface(t *testing.T) {
	var _ audit.Writer = audit.LogWriter{}
}

func TestBigQueryWriter_ImplementsWriterInterface(t *testing.T) {
	var _ audit.Writer = (*audit.BigQueryWriter)(nil)
}
