terraform {
  required_version = ">= 1.5"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# ── Service Account ──────────────────────────────────────────────────────────

resource "google_service_account" "gar_cleanup" {
  account_id   = "gar-cleanup"
  display_name = "GAR Cleanup Service Account"
  description  = "Used by the gar-cleanup Cloud Run job for image lifecycle management"
}

# Artifact Registry repo admin on specific repos (not project-wide).
resource "google_project_iam_member" "gar_cleanup_gar" {
  project = var.project_id
  role    = "roles/artifactregistry.repoAdmin"
  member  = "serviceAccount:${google_service_account.gar_cleanup.email}"
}

# Read-only K8s cluster access for the image guard.
resource "google_project_iam_member" "gar_cleanup_k8s" {
  project = var.project_id
  role    = "roles/container.clusterViewer"
  member  = "serviceAccount:${google_service_account.gar_cleanup.email}"
}

# BigQuery data editor for audit events.
resource "google_project_iam_member" "gar_cleanup_bq" {
  project = var.project_id
  role    = "roles/bigquery.dataEditor"
  member  = "serviceAccount:${google_service_account.gar_cleanup.email}"
}

# Allow Cloud Scheduler to invoke the Cloud Run job.
resource "google_project_iam_member" "gar_cleanup_run_invoker" {
  project = var.project_id
  role    = "roles/run.invoker"
  member  = "serviceAccount:${google_service_account.gar_cleanup.email}"
}

# ── Cloud Run Job ─────────────────────────────────────────────────────────────

resource "google_cloud_run_v2_job" "gar_cleanup" {
  name     = "gar-cleanup"
  location = var.region

  template {
    template {
      service_account = google_service_account.gar_cleanup.email

      containers {
        image = var.image

        args = [
          "run",
          "--project", var.project_id,
          "--location", var.region,
          "--policy", "/etc/gar-cleanup/policy.yaml",
        ]

        # Mount policy.yaml from Secret Manager (or GCS via init container).
        # For simplicity, the policy file is baked into the image; for
        # production, inject via volume or Secret Manager.
        env {
          name  = "GOOGLE_CLOUD_PROJECT"
          value = var.project_id
        }

        resources {
          limits = {
            cpu    = "1"
            memory = "512Mi"
          }
        }
      }

      timeout     = "600s"
      max_retries = 1
    }
  }

  lifecycle {
    ignore_changes = [
      launch_stage,
    ]
  }
}

# ── Cloud Scheduler ───────────────────────────────────────────────────────────

resource "google_cloud_scheduler_job" "gar_cleanup" {
  name      = "gar-cleanup-daily"
  region    = var.region
  schedule  = var.schedule
  time_zone = "UTC"

  http_target {
    uri         = "https://${var.region}-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/${var.project_id}/jobs/gar-cleanup:run"
    http_method = "POST"

    oauth_token {
      service_account_email = google_service_account.gar_cleanup.email
    }
  }
}

# ── BigQuery Audit Table ──────────────────────────────────────────────────────

resource "google_bigquery_dataset" "gar_cleanup" {
  dataset_id  = "gar_cleanup_audit"
  location    = var.bq_location
  description = "Audit events from the GAR cleanup engine"
}

resource "google_bigquery_table" "deletion_events" {
  dataset_id          = google_bigquery_dataset.gar_cleanup.dataset_id
  table_id            = "deletion_events"
  deletion_protection = false

  schema = jsonencode([
    { name = "timestamp", type = "TIMESTAMP", mode = "REQUIRED" },
    { name = "registry", type = "STRING", mode = "REQUIRED" },
    { name = "repository", type = "STRING", mode = "REQUIRED" },
    { name = "digest", type = "STRING", mode = "REQUIRED" },
    { name = "tags", type = "STRING", mode = "REPEATED" },
    { name = "size_bytes", type = "INTEGER", mode = "NULLABLE" },
    { name = "reason", type = "STRING", mode = "REQUIRED" },
    { name = "dry_run", type = "BOOL", mode = "REQUIRED" },
    { name = "k8s_protected", type = "BOOL", mode = "REQUIRED" },
  ])
}
