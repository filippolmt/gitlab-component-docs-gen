# Tests & CI Pipeline Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add unit tests, integration tests, and a CI pipeline that runs them on every push and PR.

**Architecture:** Extract the auto-creation logic from `main()` into a testable `ensureTemplate()` function. Test `parseTemplate` with temp YAML fixtures and `ensureTemplate` with temp directories. Add an integration test that builds the binary and runs it end-to-end. CI runs via a new GitHub Actions workflow on push/PR.

**Tech Stack:** Go standard `testing` package, `t.TempDir()`, `os/exec` for integration, GitHub Actions

---

### Task 1: Extract `ensureTemplate` from `main()` for testability

**Files:**
- Modify: `main.go:92-101` (extract auto-creation logic into function)

**Step 1: Add `ensureTemplate` function above `main()`**

Insert this function between `parseTemplate` (ends line 90) and `main()` (line 92):

```go
// ensureTemplate controlla se il file template esiste, se non esiste lo crea dal default
func ensureTemplate(path string, defaultContent []byte) (bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.WriteFile(path, defaultContent, 0644)
		if err != nil {
			return false, fmt.Errorf("error creating default %s: %w", path, err)
		}
		return true, nil
	}
	return false, nil
}
```

**Step 2: Update `main()` to use `ensureTemplate`**

Replace the existing auto-creation block (lines 93-101) with:

```go
func main() {
	// Se README.md.tmpl non esiste, crealo dal template di default
	created, err := ensureTemplate("README.md.tmpl", defaultTemplate)
	if err != nil {
		fmt.Println(err)
		return
	}
	if created {
		fmt.Println("Created default README.md.tmpl")
	}

	// Cerca tutti i template nella directory templates/
```

**Step 3: Verify it compiles**

Run: `cd /Users/filippomerante/project/github/gitlab_component && go build -o gitlab_component main.go`
Expected: compiles with no errors

**Step 4: Clean up binary**

Run: `rm /Users/filippomerante/project/github/gitlab_component/gitlab_component`

**Step 5: Commit**

```bash
git add main.go
git commit -m "refactor: extract ensureTemplate for testability"
```

---

### Task 2: Test `parseTemplate`

**Files:**
- Create: `main_test.go`

**Step 1: Write tests for `parseTemplate`**

Create `main_test.go` with tests covering: basic parsing, sorting (required first then alphabetical), error on missing file, and error on invalid YAML.

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTemplate_BasicParsing(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `spec:
  description: "Deploy application"
  inputs:
    app_name:
      description: "Application name"
    stage:
      description: "Pipeline stage"
      default: "deploy"
`
	path := filepath.Join(dir, "deploy.yml")
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	component, err := parseTemplate(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if component.Name != "deploy" {
		t.Errorf("expected name 'deploy', got '%s'", component.Name)
	}
	if component.Description != "Deploy application" {
		t.Errorf("expected description 'Deploy application', got '%s'", component.Description)
	}
	if len(component.Inputs) != 2 {
		t.Fatalf("expected 2 inputs, got %d", len(component.Inputs))
	}
}

func TestParseTemplate_SortingRequiredFirst(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `spec:
  description: "Test"
  inputs:
    zebra:
      description: "Optional Z"
      default: "z"
    alpha:
      description: "Required A"
    beta:
      description: "Optional B"
      default: "b"
    gamma:
      description: "Required G"
`
	path := filepath.Join(dir, "test.yml")
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	component, err := parseTemplate(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Required first (alpha, gamma), then optional (beta, zebra)
	expected := []struct {
		name     string
		required bool
	}{
		{"alpha", true},
		{"gamma", true},
		{"beta", false},
		{"zebra", false},
	}

	if len(component.Inputs) != len(expected) {
		t.Fatalf("expected %d inputs, got %d", len(expected), len(component.Inputs))
	}

	for i, exp := range expected {
		if component.Inputs[i].Name != exp.name {
			t.Errorf("input[%d]: expected name '%s', got '%s'", i, exp.name, component.Inputs[i].Name)
		}
		if component.Inputs[i].Required != exp.required {
			t.Errorf("input[%d] '%s': expected required=%v, got %v", i, exp.name, exp.required, component.Inputs[i].Required)
		}
	}
}

func TestParseTemplate_MissingFile(t *testing.T) {
	_, err := parseTemplate("/nonexistent/file.yml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestParseTemplate_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yml")
	if err := os.WriteFile(path, []byte("not: [valid: yaml: {{{}"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := parseTemplate(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestParseTemplate_NoInputs(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `spec:
  description: "Simple component"
`
	path := filepath.Join(dir, "simple.yml")
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	component, err := parseTemplate(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if component.Name != "simple" {
		t.Errorf("expected name 'simple', got '%s'", component.Name)
	}
	if len(component.Inputs) != 0 {
		t.Errorf("expected 0 inputs, got %d", len(component.Inputs))
	}
}
```

**Step 2: Run tests to verify they pass**

Run: `cd /Users/filippomerante/project/github/gitlab_component && go test -v -run TestParseTemplate`
Expected: all 5 tests PASS

**Step 3: Commit**

```bash
git add main_test.go
git commit -m "test: add unit tests for parseTemplate"
```

---

### Task 3: Test `ensureTemplate`

**Files:**
- Modify: `main_test.go` (append tests)

**Step 1: Add tests for `ensureTemplate`**

Append to `main_test.go`:

```go
func TestEnsureTemplate_CreatesWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md.tmpl")
	content := []byte("template content")

	created, err := ensureTemplate(path, content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected created=true, got false")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}
	if string(data) != "template content" {
		t.Errorf("expected 'template content', got '%s'", string(data))
	}
}

func TestEnsureTemplate_SkipsWhenExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md.tmpl")
	original := []byte("original content")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}

	created, err := ensureTemplate(path, []byte("new content"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Error("expected created=false, got true")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != "original content" {
		t.Errorf("expected file to keep 'original content', got '%s'", string(data))
	}
}
```

**Step 2: Run tests to verify they pass**

Run: `cd /Users/filippomerante/project/github/gitlab_component && go test -v -run TestEnsureTemplate`
Expected: both tests PASS

**Step 3: Commit**

```bash
git add main_test.go
git commit -m "test: add unit tests for ensureTemplate"
```

---

### Task 4: Integration test (end-to-end)

**Files:**
- Modify: `main_test.go` (append integration test)

**Step 1: Add integration test**

Append to `main_test.go`. This test builds the binary and runs it in a temp directory with test fixtures:

```go
import (
	// add to existing imports:
	"os/exec"
	"strings"
)

func TestIntegration_GeneratesREADME(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build the binary
	binary := filepath.Join(t.TempDir(), "gitlab_component")
	build := exec.Command("go", "build", "-o", binary, "main.go")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	// Set up a test working directory
	workDir := t.TempDir()
	templatesDir := filepath.Join(workDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatal(err)
	}

	yamlContent := `spec:
  description: "Build application"
  inputs:
    app_name:
      description: "Application name"
    stage:
      description: "Pipeline stage"
      default: "build"
`
	if err := os.WriteFile(filepath.Join(templatesDir, "build.yml"), []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Run the binary (no README.md.tmpl — should auto-create it)
	cmd := exec.Command(binary)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, out)
	}

	output := string(out)
	if !strings.Contains(output, "Created default README.md.tmpl") {
		t.Error("expected 'Created default README.md.tmpl' in output")
	}
	if !strings.Contains(output, "Documentation generated successfully!") {
		t.Error("expected 'Documentation generated successfully!' in output")
	}

	// Verify README.md was generated
	readme, err := os.ReadFile(filepath.Join(workDir, "README.md"))
	if err != nil {
		t.Fatalf("README.md not created: %v", err)
	}
	if !strings.Contains(string(readme), "build") {
		t.Error("expected README.md to contain component name 'build'")
	}
	if !strings.Contains(string(readme), "Application name") {
		t.Error("expected README.md to contain input description")
	}

	// Verify README.md.tmpl was auto-created
	if _, err := os.Stat(filepath.Join(workDir, "README.md.tmpl")); os.IsNotExist(err) {
		t.Error("expected README.md.tmpl to be auto-created")
	}
}
```

Note: the additional imports (`os/exec`, `strings`) must be added to the existing import block at the top of `main_test.go`.

**Step 2: Run the integration test**

Run: `cd /Users/filippomerante/project/github/gitlab_component && go test -v -run TestIntegration`
Expected: PASS

**Step 3: Run all tests together**

Run: `cd /Users/filippomerante/project/github/gitlab_component && go test -v ./...`
Expected: all tests PASS

**Step 4: Commit**

```bash
git add main_test.go
git commit -m "test: add integration test for end-to-end README generation"
```

---

### Task 5: Create CI workflow

**Files:**
- Create: `.github/workflows/ci.yml`

**Step 1: Create the CI workflow**

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Build
        run: go build -o gitlab_component main.go

      - name: Test
        run: go test -v ./...
```

**Step 2: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add CI workflow with build and test on push/PR"
```

---

### Task 6: Update CLAUDE.md

**Files:**
- Modify: `CLAUDE.md`

**Step 1: Update the "Build & Run" section**

Replace the line "There are no tests or linting configured in this project." with:

```markdown
## Tests

```bash
# Run all tests
go test -v ./...

# Run unit tests only (skip integration)
go test -v -short ./...
```
```

**Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md with test commands"
```
