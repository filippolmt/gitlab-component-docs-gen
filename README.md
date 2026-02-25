# gitlab-component-docs-gen

A Go utility that automatically generates README documentation from [GitLab CI/CD Component](https://docs.gitlab.com/ee/ci/components/) template specs.

## What it does

It scans all `.yml` files in a `templates/` directory, parses the `spec` section of each GitLab CI/CD component, and generates a `README.md` with:

- A section per component (derived from the filename)
- An optional description per component (from `docs/<name>.md`)
- A usage example with the correct component path and version
- An inputs table with name, description, required flag, and default value

Inputs are sorted with required parameters first, then alphabetically.

## Requirements

Each template YAML must have a `spec` section following the [GitLab CI/CD component spec](https://docs.gitlab.com/ee/ci/components/#spec) format:

```yaml
spec:
  inputs:
    app_name:
      description: "The name of the application"
    stage:
      description: "The CI/CD stage"
      default: "build"
---
# ... rest of the CI/CD job definition
```

- Inputs **without** a `default` are marked as required

## Usage

### With Docker (recommended)

Run from the root of your GitLab CI/CD component repository:

```bash
docker run --rm -v $(pwd):/app ghcr.io/<owner>/gitlab-component-docs-gen
```

If `README.md.tmpl` doesn't exist, it will be auto-created from the embedded default template.

The image is published to `ghcr.io` on every push to main (tagged with commit SHA) and on version tags (tagged with semver + `latest`). Old SHA-tagged images are cleaned up weekly.

### From source

```bash
go run main.go
```

### CLI flags

```bash
gitlab-component-docs-gen --project-path group/project --version 1.0.0
```

| Flag | Env var | Description |
|------|---------|-------------|
| `--project-path` | `PROJECT_PATH` | GitLab project path (e.g. `group/project`) |
| `--version` | `VERSION` | Component version (e.g. `1.0.0`) |

Both values are resolved with this priority: **flag > env var > config file > git auto-detect > placeholder**.

- **Project path** auto-detects from `git remote get-url origin` (SSH and HTTPS)
- **Version** auto-detects from `git describe --tags --abbrev=0`

## Configuration

Create an optional `.gitlab-component-docs-gen.yml` in the repository root:

```yaml
project_path: my-group/my-project
version: "1.0.0"
```

## Component descriptions

To add a custom description for a component, create a markdown file in `docs/` matching the component name:

```
templates/build.yml   →  docs/build.md
templates/deploy.yml  →  docs/deploy.md
```

The content of `docs/<name>.md` is inserted in the generated README between the usage example and the inputs table. If the file doesn't exist, no description is shown.

## Customizing the template

The `README.md.tmpl` file uses Go's `text/template` syntax. Available data:

```
.ProjectPath            - Resolved project path
.Version                - Resolved version
.Components[]
  .Name                 - Component name (filename without .yml extension)
  .Description          - Content of docs/<name>.md (empty if missing)
  .Inputs[]
    .Name               - Input parameter name
    .Description        - Input description
    .Required           - true if no default is set
    .Default            - Default value (empty string if required)
```

## License

GPL-3.0 - see [LICENSE](LICENSE) for details.
