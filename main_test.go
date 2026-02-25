package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTemplate_BasicParsing(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `spec:
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
	if len(component.Inputs) != 2 {
		t.Fatalf("expected 2 inputs, got %d", len(component.Inputs))
	}
}

func TestParseTemplate_SortingRequiredFirst(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `spec:
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
  inputs: {}
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

func TestLoadComponentDescription_Exists(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	os.MkdirAll(docsDir, 0755)
	os.WriteFile(filepath.Join(docsDir, "build.md"), []byte("This component builds your app.\n"), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	got := loadComponentDescription("build")
	if got != "This component builds your app." {
		t.Errorf("expected 'This component builds your app.', got %q", got)
	}
}

func TestLoadComponentDescription_Missing(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	got := loadComponentDescription("nonexistent")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestParseTemplate_WithDescription(t *testing.T) {
	dir := t.TempDir()

	// Create docs directory with description
	docsDir := filepath.Join(dir, "docs")
	os.MkdirAll(docsDir, 0755)
	os.WriteFile(filepath.Join(docsDir, "deploy.md"), []byte("Deploys the application to production."), 0644)

	// Create template YAML
	yamlContent := `spec:
  inputs:
    app_name:
      description: "Application name"
`
	path := filepath.Join(dir, "deploy.yml")
	os.WriteFile(path, []byte(yamlContent), 0644)

	// Change to dir so loadComponentDescription finds docs/
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	component, err := parseTemplate(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if component.Description != "Deploys the application to production." {
		t.Errorf("expected description 'Deploys the application to production.', got %q", component.Description)
	}
}

func TestFormatDefault(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"nil", nil, ""},
		{"string", "deploy", "deploy"},
		{"empty string", "", ""},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"array", []interface{}{map[string]interface{}{"if": "$CI_COMMIT_BRANCH == \"main\""}}, "`[{\"if\":\"$CI_COMMIT_BRANCH == \\\"main\\\"\"}]`"},
		{"map", map[string]interface{}{"key": "value"}, "`{\"key\":\"value\"}`"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDefault(tt.input)
			if got != tt.expected {
				t.Errorf("formatDefault(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseTemplate_ComplexDefaults(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `spec:
  inputs:
    app_name:
      description: "Application name"
    rules:
      description: "Execution rules"
      type: array
      default:
        - if: '$CI_COMMIT_BRANCH == "main"'
    save_artifacts:
      description: "Save artifacts"
      default: true
      type: boolean
    timeout:
      description: "Timeout"
      default: "5m0s"
`
	path := filepath.Join(dir, "deploy.yml")
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	component, err := parseTemplate(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(component.Inputs) != 4 {
		t.Fatalf("expected 4 inputs, got %d", len(component.Inputs))
	}

	// app_name is required (no default), should be first
	if component.Inputs[0].Name != "app_name" || !component.Inputs[0].Required {
		t.Errorf("expected first input to be required 'app_name', got '%s' required=%v", component.Inputs[0].Name, component.Inputs[0].Required)
	}

	// Find the other inputs by name
	byName := make(map[string]InputData)
	for _, input := range component.Inputs {
		byName[input.Name] = input
	}

	// rules: array default should be JSON-serialized
	rules := byName["rules"]
	if rules.Required {
		t.Error("rules should not be required (has default)")
	}
	if !strings.Contains(rules.Default, "CI_COMMIT_BRANCH") {
		t.Errorf("rules default should contain CI_COMMIT_BRANCH, got %q", rules.Default)
	}

	// save_artifacts: boolean default
	sa := byName["save_artifacts"]
	if sa.Required {
		t.Error("save_artifacts should not be required (has default)")
	}
	if sa.Default != "true" {
		t.Errorf("save_artifacts default should be 'true', got %q", sa.Default)
	}

	// timeout: string default
	timeout := byName["timeout"]
	if timeout.Required {
		t.Error("timeout should not be required (has default)")
	}
	if timeout.Default != "5m0s" {
		t.Errorf("timeout default should be '5m0s', got %q", timeout.Default)
	}
}

func TestParseGitRemoteURL(t *testing.T) {
	tests := []struct {
		name     string
		remote   string
		expected string
	}{
		{"SSH", "git@gitlab.com:group/project.git", "group/project"},
		{"SSH with subgroup", "git@gitlab.com:group/subgroup/project.git", "group/subgroup/project"},
		{"SSH without .git", "git@gitlab.com:group/project", "group/project"},
		{"HTTPS", "https://gitlab.com/group/project.git", "group/project"},
		{"HTTPS with subgroup", "https://gitlab.com/group/subgroup/project.git", "group/subgroup/project"},
		{"HTTPS without .git", "https://gitlab.com/group/project", "group/project"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGitRemoteURL(tt.remote)
			if got != tt.expected {
				t.Errorf("parseGitRemoteURL(%q) = %q, want %q", tt.remote, got, tt.expected)
			}
		})
	}
}

func TestReadConfigProjectPath(t *testing.T) {
	dir := t.TempDir()
	configContent := `project_path: my-group/my-project
`
	if err := os.WriteFile(filepath.Join(dir, ".gitlab-component-docs-gen.yml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to temp dir to test config reading
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	got := readConfigProjectPath()
	if got != "my-group/my-project" {
		t.Errorf("expected 'my-group/my-project', got %q", got)
	}
}

func TestReadConfigProjectPath_Missing(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	got := readConfigProjectPath()
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestResolveVersion_Priority(t *testing.T) {
	// Set up config file with version in temp dir
	dir := t.TempDir()
	configContent := `version: "2.0.0"
`
	os.WriteFile(filepath.Join(dir, ".gitlab-component-docs-gen.yml"), []byte(configContent), 0644)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Flag takes priority over everything
	got := resolveVersion("1.0.0")
	if got != "1.0.0" {
		t.Errorf("expected '1.0.0', got %q", got)
	}

	// Env var takes priority over config
	t.Setenv("VERSION", "1.5.0")
	got = resolveVersion("")
	if got != "1.5.0" {
		t.Errorf("expected '1.5.0', got %q", got)
	}

	// Config takes priority when no flag or env
	os.Unsetenv("VERSION")
	got = resolveVersion("")
	if got != "2.0.0" {
		t.Errorf("expected '2.0.0', got %q", got)
	}
}

func TestResolveVersion_Fallback(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	os.Unsetenv("VERSION")
	got := resolveVersion("")
	if got != "<version>" {
		t.Errorf("expected '<version>', got %q", got)
	}
}

func TestResolveProjectPath_Priority(t *testing.T) {
	// Set up config file in temp dir
	dir := t.TempDir()
	configContent := `project_path: from-config
`
	os.WriteFile(filepath.Join(dir, ".gitlab-component-docs-gen.yml"), []byte(configContent), 0644)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Flag takes priority over everything
	got := resolveProjectPath("from-flag")
	if got != "from-flag" {
		t.Errorf("expected 'from-flag', got %q", got)
	}

	// Env var takes priority over config
	t.Setenv("PROJECT_PATH", "from-env")
	got = resolveProjectPath("")
	if got != "from-env" {
		t.Errorf("expected 'from-env', got %q", got)
	}

	// Config takes priority when no flag or env
	os.Unsetenv("PROJECT_PATH")
	got = resolveProjectPath("")
	if got != "from-config" {
		t.Errorf("expected 'from-config', got %q", got)
	}
}

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

func TestIntegration_GeneratesREADME(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build the binary
	binary := filepath.Join(t.TempDir(), "gitlab-component-docs-gen")
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
