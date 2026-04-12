output "service_account_email" {
  description = "Email of the gar-cleanup service account"
  value       = google_service_account.gar_cleanup.email
}

output "cloud_run_job_name" {
  description = "Name of the Cloud Run job"
  value       = google_cloud_run_v2_job.gar_cleanup.name
}

output "scheduler_job_name" {
  description = "Name of the Cloud Scheduler job"
  value       = google_cloud_scheduler_job.gar_cleanup.name
}

output "bigquery_table" {
  description = "Full BigQuery table ID for audit events"
  value       = "${var.project_id}.${google_bigquery_dataset.gar_cleanup.dataset_id}.${google_bigquery_table.deletion_events.table_id}"
}
