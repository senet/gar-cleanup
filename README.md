# gar-cleanup

A policy-driven container image lifecycle manager for **Google Artifact Registry (GAR)**. Replaces the deprecated `gcr.io` shell script with a production-grade Go binary that runs as a scheduled Cloud Run Job.

[![CI](https://github.com/senet/gcr-cleanup/actions/workflows/ci.yml/badge.svg)](https://github.com/senet/gcr-cleanup/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

## Features

- **Policy-as-code** — retention rules in `policy.yaml`, reviewed via pull request
- **Dry-run by default** — see what would be deleted before committing
- **Kubernetes guard** — never deletes images in use by running workloads *(phase 2)*
- **BigQuery audit trail** — every deletion recorded for cost attribution *(phase 4)*
- **Semver protection** — `v1.2.3`-style tags are never deleted unless explicitly allowed
- **Terraform IaC** — Cloud Run Job + Scheduler + BigQuery table ready to deploy

## Quick Start

```bash
# Build
make build

# Dry-run against your project (reads policy.yaml)
./bin/gar-cleanup run \
  --project my-gcp-project \
  --location us-central1 \
  --policy policy.yaml

# Live deletion (requires dry_run: false in policy.yaml)
./bin/gar-cleanup run \
  --project my-gcp-project \
  --live
```

## Policy File

Edit `policy.yaml` to configure retention rules:

```yaml
dry_run: true          # set false only after reviewing the dry-run log

keep: 10               # retain the 10 most-recently pushed images per repo
max_age: 30d           # delete images older than 30 days

protect_tags:          # these tags are NEVER deleted
  - "^v\\d+\\.\\d+\\.\\d+$"   # semver releases
  - "^stable$"
  - "^production$"

delete_untagged: true  # delete digests with no tags immediately
```

Policy changes must go through a pull request — the policy is applied on the next scheduled run.

## Development

**Requirements:** Go 1.24+

```bash
# Build
make build

# Run tests
make test

# Coverage report
make cover
```

## Deployment

Infrastructure is managed with Terraform:

```bash
cd terraform
terraform init
terraform apply -var="project_id=my-gcp-project"
```

This creates:
- A `gar-cleanup` service account with least-privilege IAM bindings
- A Cloud Run Job that executes the cleanup binary
- A Cloud Scheduler job that triggers it daily at 02:00 UTC
- A BigQuery table for audit events

See [ARCHITECTURE.md](ARCHITECTURE.md) for full design documentation.

## Commands

| Command | Description |
|---------|-------------|
| `gar-cleanup run` | Execute a cleanup run (dry-run by default) |
| `gar-cleanup version` | Print version, commit, and build date |

### `run` flags

| Flag | Default | Description |
|------|---------|-------------|
| `--policy` | `policy.yaml` | Path to YAML policy file |
| `--project` | `$GOOGLE_CLOUD_PROJECT` | GCP project ID |
| `--location` | `us-central1` | GAR location |
| `--live` | `false` | Execute live deletions |

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for the full design, component map, and delivery phases.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

To report a vulnerability, see [.github/SECURITY.md](.github/SECURITY.md). **Do not open a public issue.**

## License

Apache 2.0 — see [LICENSE](LICENSE).
