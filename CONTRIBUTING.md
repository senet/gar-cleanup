# Contributing to gar-cleanup

Thank you for your interest in contributing to gar-cleanup!

## Development Setup

1. Install Go 1.24+
2. Clone the repo:
   ```bash
   git clone https://github.com/senet/gar-cleanup.git
   cd gar-cleanup
   ```
3. Build:
   ```bash
   make build
   ```
4. Run tests:
   ```bash
   make test
   ```

## Code Style

- Run `make lint` before submitting PRs
- Follow standard Go conventions
- Add tests for new functionality

## Pull Requests

1. Fork the repo
2. Create a feature branch from `main`
3. Add tests for your changes
4. Ensure `make test` and `make lint` pass
5. Submit a PR with a clear description

All PRs require:
- Passing CI (lint + test + build)
- Approval from the repository owner ([@senet](https://github.com/senet))
- Review from a [CODEOWNER](/.github/CODEOWNERS)

## Policy Changes

Policy changes (`policy.yaml`) are code changes — they must go through a PR,
be reviewed, and are applied on the next scheduled run.

## Branch Protection

The `main` branch is protected:
- Direct pushes are not allowed
- All changes must go through a pull request
- CI status checks must pass before merging
- Force pushes and branch deletion are blocked

## Reporting Issues

Open an issue on GitHub with:
- A clear description of the problem
- Steps to reproduce
- Expected vs actual behaviour
- gar-cleanup version (`gar-cleanup version`)

## Security

To report a security vulnerability, see [SECURITY.md](/.github/SECURITY.md).
**Do not open a public issue for security vulnerabilities.**
