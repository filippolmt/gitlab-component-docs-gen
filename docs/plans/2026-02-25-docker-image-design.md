# Docker Image for gitlab_component

## Goal

Publish a standalone Docker image on ghcr.io that generates README.md from GitLab CI/CD component specs. On first run, if `README.md.tmpl` doesn't exist, the tool creates it with a sensible default.

## Architecture

Multi-stage Dockerfile:
- **Builder stage:** `golang:alpine` — compiles a static binary (CGO_DISABLED=1)
- **Final stage:** `scratch` — contains only the compiled binary

## Runtime behavior

User mounts the repo root to `/app`:

```bash
docker run -v $(pwd):/app ghcr.io/<owner>/gitlab_component
```

1. If `README.md.tmpl` does not exist in `/app`, create it from the embedded default template
2. Read `templates/*.yml`, generate `README.md` using the template

The default template is embedded in the Go binary via `//go:embed`.

## Changes to main.go

- Add `embed` package to embed `README.md.tmpl` as default
- Before parsing, check if `README.md.tmpl` exists; if not, write the embedded default
- Working directory is `/app` (set via Dockerfile WORKDIR)

## Publishing

- Registry: ghcr.io
- GitHub Actions workflow triggers on tag push (`v*`)
- Image tags: `latest` + version from tag (e.g., `v1.0.0` → `1.0.0`)

## New files

- `Dockerfile` — multi-stage build, scratch final image
- `.github/workflows/docker-publish.yml` — build and push on tag
