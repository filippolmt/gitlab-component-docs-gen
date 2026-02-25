# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A single-file Go CLI tool that generates README.md documentation from GitLab CI/CD Component template specs. It parses `spec` sections from YAML files in `templates/*.yml` and renders a `README.md` using a Go `text/template` in `README.md.tmpl`.

## Build & Run

```bash
# Run the tool (reads templates/*.yml, writes README.md)
go run main.go

# Build a binary
go build -o gitlab_component main.go
```

There are no tests or linting configured in this project.

## Architecture

Everything lives in `main.go` — a single-pass pipeline:

1. **Glob** `templates/*.yml` (sorted alphabetically)
2. **Parse** each YAML file's `spec` section into `Config` → `ComponentData` structs using `goccy/go-yaml`
3. **Sort** inputs: required first, then alphabetically by name
4. **Render** `README.md.tmpl` with the collected `TemplateData{Components}` and write `README.md`

Key types: `Config` (YAML structure) → `ComponentData`/`InputData` (template data). An input is "required" when its `default` field is empty string.

## Key Files

- `main.go` — all logic (parsing, sorting, rendering)
- `README.md.tmpl` — Go text/template that defines the generated README format
- `README.md` — **generated output**, not manually edited (will be overwritten on each run)

## Conventions

- The Go module is named `doc` (in `go.mod`)
- Code comments are in Italian
- The tool expects to be run from the repository root where `templates/` and `README.md.tmpl` exist
