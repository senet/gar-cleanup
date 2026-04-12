variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region for Cloud Run and Scheduler"
  type        = string
  default     = "us-central1"
}

variable "image" {
  description = "Container image for the gar-cleanup Cloud Run job"
  type        = string
  default     = "us-central1-docker.pkg.dev/PROJECT/gar-cleanup/gar-cleanup:latest"
}

variable "schedule" {
  description = "Cron expression for Cloud Scheduler (UTC)"
  type        = string
  default     = "0 2 * * *"
}

variable "bq_location" {
  description = "BigQuery dataset location"
  type        = string
  default     = "US"
}
