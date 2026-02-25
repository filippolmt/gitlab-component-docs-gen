# Docker Image Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Package gitlab_component as a scratch-based Docker image published to ghcr.io, with auto-creation of README.md.tmpl on first run.

**Architecture:** Embed the default template in the Go binary via `//go:embed`. At startup, check if `README.md.tmpl` exists — if not, write the embedded default. Multi-stage Dockerfile compiles a static binary, copies it to scratch. GitHub Actions workflow builds and pushes the image on version tags.

**Tech Stack:** Go embed, Docker multi-stage (golang:alpine → scratch), GitHub Actions, ghcr.io

---

### Task 1: Embed default template in Go binary

**Files:**
- Modify: `main.go:1-12` (imports and add embed directive)
- Existing: `README.md.tmpl` (used as embed source)

**Step 1: Add embed import and directive to main.go**

Add `embed` to the import block and add the embed directive above a new variable. Place this right after the import block (after line 12):

```go
import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/template"

	"github.com/goccy/go-yaml"
)

//go:embed README.md.tmpl
var defaultTemplate []byte
```

**Step 2: Verify it compiles**

Run: `cd /Users/filippomerante/project/github/gitlab_component && go build -o gitlab_component main.go`
Expected: compiles with no errors

**Step 3: Clean up binary**

Run: `rm /Users/filippomerante/project/github/gitlab_component/gitlab_component`

**Step 4: Commit**

```bash
git add main.go
git commit -m "feat: embed default README.md.tmpl in binary"
```

---

### Task 2: Add auto-creation of README.md.tmpl

**Files:**
- Modify: `main.go:88-99` (beginning of main function)

**Step 1: Add template auto-creation logic at the start of main()**

Insert this block at the beginning of `main()`, before the `filepath.Glob` call:

```go
func main() {
	// Se README.md.tmpl non esiste, crealo dal template di default
	if _, err := os.Stat("README.md.tmpl"); os.IsNotExist(err) {
		err = os.WriteFile("README.md.tmpl", defaultTemplate, 0644)
		if err != nil {
			fmt.Printf("Error creating default README.md.tmpl: %s\n", err)
			return
		}
		fmt.Println("Created default README.md.tmpl")
	}

	// Cerca tutti i template nella directory templates/
```

**Step 2: Verify it compiles**

Run: `cd /Users/filippomerante/project/github/gitlab_component && go build -o gitlab_component main.go`
Expected: compiles with no errors

**Step 3: Test the auto-creation behavior**

Run:
```bash
cd /tmp && mkdir -p test-gitlab-component/templates
# Create a dummy template YAML
cat > test-gitlab-component/templates/test.yml << 'EOF'
spec:
  description: "Test component"
  inputs:
    app_name:
      description: "App name"
    stage:
      description: "Stage"
      default: "build"
---
test_job:
  script: echo hello
EOF
# Copy the binary
cp /Users/filippomerante/project/github/gitlab_component/gitlab_component test-gitlab-component/
cd test-gitlab-component && ./gitlab_component
```

Expected: prints "Created default README.md.tmpl" then "Documentation generated successfully!", and both `README.md.tmpl` and `README.md` exist.

**Step 4: Clean up**

Run: `rm -rf /tmp/test-gitlab-component && rm /Users/filippomerante/project/github/gitlab_component/gitlab_component`

**Step 5: Commit**

```bash
git add main.go
git commit -m "feat: auto-create README.md.tmpl from embedded default on first run"
```

---

### Task 3: Create Dockerfile

**Files:**
- Create: `Dockerfile`

**Step 1: Create the multi-stage Dockerfile**

```dockerfile
FROM golang:alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY main.go README.md.tmpl ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o gitlab_component main.go

FROM scratch
COPY --from=builder /build/gitlab_component /gitlab_component
WORKDIR /app
ENTRYPOINT ["/gitlab_component"]
```

**Step 2: Verify Docker build succeeds**

Run: `cd /Users/filippomerante/project/github/gitlab_component && docker build -t gitlab_component:test .`
Expected: builds successfully, final image is scratch-based

**Step 3: Test Docker run with auto-creation**

Run:
```bash
cd /tmp && mkdir -p docker-test/templates
cat > docker-test/templates/test.yml << 'EOF'
spec:
  description: "Test component"
  inputs:
    app_name:
      description: "App name"
EOF
docker run --rm -v /tmp/docker-test:/app gitlab_component:test
```

Expected: prints "Created default README.md.tmpl" and "Documentation generated successfully!". Verify files exist: `ls /tmp/docker-test/README.md /tmp/docker-test/README.md.tmpl`

**Step 4: Test Docker run with existing template**

Run:
```bash
docker run --rm -v /tmp/docker-test:/app gitlab_component:test
```

Expected: prints only "Documentation generated successfully!" (no "Created default" message since template already exists)

**Step 5: Clean up**

Run: `rm -rf /tmp/docker-test && docker rmi gitlab_component:test`

**Step 6: Commit**

```bash
git add Dockerfile
git commit -m "feat: add multi-stage Dockerfile with scratch final image"
```

---

### Task 4: Create GitHub Actions workflow

**Files:**
- Create: `.github/workflows/docker-publish.yml`

**Step 1: Create the workflow file**

```yaml
name: Build and Push Docker Image

on:
  push:
    tags:
      - 'v*'

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  build-and-push:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Log in to Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=semver,pattern={{version}}
            type=raw,value=latest

      - name: Build and push
        uses: docker/build-push-action@v6
        with:
          context: .
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
```

**Step 2: Commit**

```bash
git add .github/workflows/docker-publish.yml
git commit -m "ci: add GitHub Actions workflow for Docker image publishing"
```

---

### Task 5: Update documentation

**Files:**
- Modify: `README.md` (not the generated one — but since this IS generated, we update `README.md.tmpl` or leave usage in README as-is)
- Modify: `CLAUDE.md`

**Step 1: Update CLAUDE.md with Docker info**

Add a Docker section to `CLAUDE.md` covering:
- `docker build -t gitlab_component .`
- `docker run -v $(pwd):/app gitlab_component`
- ghcr.io publishing triggered by version tags

**Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md with Docker build and run instructions"
```
