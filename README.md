# gitlab-component-docs-gen

A Go utility that automatically generates README documentation from [GitLab CI/CD Component](https://docs.gitlab.com/ee/ci/components/) template specs.

## What it does

It scans all `.yml` files in a `templates/` directory, parses the `spec` section of each GitLab CI/CD component, and generates a `README.md` with:

- A section per component (derived from the filename)
- A usage example with the correct component path
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

### From source

1. Copy `main.go`, `go.mod`, `go.sum`, and `README.md.tmpl` into your GitLab CI/CD component repository
2. Customize `README.md.tmpl` to match your project (header text, usage examples, additional sections)
3. Run:

```bash
go run main.go
```

This will read all `templates/*.yml` and generate `README.md` from `README.md.tmpl`.

## Customizing the template

The `README.md.tmpl` file uses Go's `text/template` syntax. Available data:

```
.Components[]
  .Name         - Component name (filename without .yml extension)
  .Inputs[]
    .Name        - Input parameter name
    .Description - Input description
    .Required    - true if no default is set
    .Default     - Default value (empty string if required)
```

## License

GPL-3.0 - see [LICENSE](LICENSE) for details.
