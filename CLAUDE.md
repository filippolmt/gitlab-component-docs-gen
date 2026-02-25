# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A single-file Go CLI tool that generates README.md documentation from GitLab CI/CD Component template specs. It parses `spec` sections from YAML files in `templates/*.yml` and renders a `README.md` using a Go `text/template` in `README.md.tmpl`.

## Build & Run

```bash
# Run the tool (reads templates/*.yml, writes README.md)
go run main.go

# Build a binary
go build -o gitlab-component-docs-gen main.go

# With flags
gitlab-component-docs-gen --project-path group/project --version 1.0.0
```

## Tests

```bash
# Run all tests
go test -v ./...

# Run unit tests only (skip integration)
go test -v -short ./...

# Using Makefile
make build
make test
make clean
```

## Architecture

Everything lives in `main.go` — a single-pass pipeline:

1. **Ensure template** — if `README.md.tmpl` doesn't exist, create it from the embedded default
2. **Glob** `templates/*.yml` (sorted alphabetically)
3. **Parse** each YAML file's `spec` section into `Config` → `ComponentData` structs using `goccy/go-yaml`
4. **Load descriptions** — read optional `docs/<name>.md` for each component
5. **Sort** inputs: required first, then alphabetically by name
6. **Resolve** project path and version (flag > env > config > git > placeholder)
7. **Render** `README.md.tmpl` with the collected `TemplateData` and write `README.md`

Key types: `Config` (YAML structure) → `ComponentData`/`InputData` (template data). An input is "required" when its `default` field is `nil`.

## Key Files

- `main.go` — all logic (parsing, sorting, rendering, config resolution)
- `main_test.go` — unit and integration tests
- `README.md.tmpl` — Go text/template that defines the generated README format
- `README.md` — **generated output**, not manually edited (will be overwritten on each run)
- `.gitlab-component-docs-gen.yml` — optional config file (project_path, version)
- `docs/<name>.md` — optional per-component descriptions
- `Makefile` — build, test, clean targets
- `Dockerfile` — multi-stage build (golang:alpine → scratch)

## Docker

```bash
# Build the image locally
docker build -t gitlab-component-docs-gen .

# Run against a repository (mount the repo root as /app)
docker run --rm -v $(pwd):/app gitlab-component-docs-gen

# With env vars
docker run --rm -v $(pwd):/app -e PROJECT_PATH=group/project -e VERSION=1.0.0 gitlab-component-docs-gen
```

If `README.md.tmpl` doesn't exist in the mounted directory, the container auto-creates it from the embedded default template.

The image is published to `ghcr.io` via GitHub Actions (multiarch: amd64 + arm64):
- **Push to main** → tagged with commit SHA (e.g. `abc1234`)
- **Push tag `v*`** → tagged with semver + `latest` (e.g. `1.0.0` + `latest`)
- Old SHA-tagged versions are cleaned up weekly, keeping the last 3. Semver and `latest` tags are never deleted.

## Conventions

- The Go module is named `doc` (in `go.mod`)
- Code comments are in English
- The tool expects to be run from the repository root where `templates/` and `README.md.tmpl` exist
