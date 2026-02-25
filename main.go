package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/goccy/go-yaml"
)

//go:embed README.md.tmpl
var defaultTemplate []byte

// Struct representing YAML inputs
type Inputs struct {
	Description string      `yaml:"description"`
	Default     interface{} `yaml:"default"`
}

type Spec struct {
	Inputs map[string]Inputs `yaml:"inputs"`
}

type Config struct {
	Spec Spec `yaml:"spec"`
}

// Struct representing template data
type InputData struct {
	Name        string
	Description string
	Required    bool
	Default     string
}

type ComponentData struct {
	Name        string
	Description string
	Inputs      []InputData
}

type TemplateData struct {
	ProjectPath string
	Version     string
	Components  []ComponentData
}

// ProjectConfig represents the optional config file .gitlab-component-docs-gen.yml
type ProjectConfig struct {
	ProjectPath string `yaml:"project_path"`
	Version     string `yaml:"version"`
}

// resolveProjectPath determines the project path using priority:
// 1. CLI flag --project-path
// 2. Env var PROJECT_PATH
// 3. Config file .gitlab-component-docs-gen.yml
// 4. Git remote auto-detect
// 5. Fallback placeholder
func resolveProjectPath(flagValue string) string {
	// 1. CLI flag
	if flagValue != "" {
		return flagValue
	}

	// 2. Env var
	if envPath := os.Getenv("PROJECT_PATH"); envPath != "" {
		return envPath
	}

	// 3. Config file
	if configPath := readConfigProjectPath(); configPath != "" {
		return configPath
	}

	// 4. Git remote
	if gitPath := detectGitProjectPath(); gitPath != "" {
		return gitPath
	}

	// 5. Fallback
	return "<your-project-path>"
}

func readConfigProjectPath() string {
	data, err := os.ReadFile(".gitlab-component-docs-gen.yml")
	if err != nil {
		return ""
	}
	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return ""
	}
	return config.ProjectPath
}

func detectGitProjectPath() string {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	return parseGitRemoteURL(strings.TrimSpace(string(out)))
}

func parseGitRemoteURL(remote string) string {
	// SSH: git@gitlab.com:group/project.git
	if strings.Contains(remote, ":") && strings.Contains(remote, "@") {
		parts := strings.SplitN(remote, ":", 2)
		if len(parts) == 2 {
			path := parts[1]
			path = strings.TrimSuffix(path, ".git")
			return path
		}
	}

	// HTTPS: https://gitlab.com/group/project.git
	if strings.Contains(remote, "//") {
		parts := strings.SplitN(remote, "//", 2)
		if len(parts) == 2 {
			// Remove host: gitlab.com/group/project.git -> group/project.git
			slashIdx := strings.Index(parts[1], "/")
			if slashIdx >= 0 {
				path := parts[1][slashIdx+1:]
				path = strings.TrimSuffix(path, ".git")
				return path
			}
		}
	}

	return ""
}

// resolveVersion determines the version using priority:
// 1. CLI flag --version
// 2. Env var VERSION
// 3. Config file .gitlab-component-docs-gen.yml
// 4. Git tag auto-detect
// 5. Fallback placeholder
func resolveVersion(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}

	if envVersion := os.Getenv("VERSION"); envVersion != "" {
		return envVersion
	}

	if configVersion := readConfigVersion(); configVersion != "" {
		return configVersion
	}

	if gitVersion := detectGitVersion(); gitVersion != "" {
		return gitVersion
	}

	return "<version>"
}

func readConfigVersion() string {
	data, err := os.ReadFile(".gitlab-component-docs-gen.yml")
	if err != nil {
		return ""
	}
	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return ""
	}
	return config.Version
}

func detectGitVersion() string {
	out, err := exec.Command("git", "describe", "--tags", "--abbrev=0").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// formatDefault converts a default value to its string representation for documentation.
func formatDefault(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case bool:
		return fmt.Sprintf("%v", v)
	case []interface{}, map[string]interface{}:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return "`" + string(jsonBytes) + "`"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// loadComponentDescription reads an optional docs/<name>.md file for a component
func loadComponentDescription(name string) string {
	data, err := os.ReadFile(filepath.Join("docs", name+".md"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func parseTemplate(path string) (ComponentData, error) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return ComponentData{}, fmt.Errorf("error reading YAML file %s: %w", path, err)
	}

	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return ComponentData{}, fmt.Errorf("error parsing YAML file %s: %w", path, err)
	}

	var inputs []InputData
	for name, input := range config.Spec.Inputs {
		inputs = append(inputs, InputData{
			Name:        name,
			Description: input.Description,
			Required:    input.Default == nil,
			Default:     formatDefault(input.Default),
		})
	}

	sort.Slice(inputs, func(i, j int) bool {
		// Sort required first, then alphabetically by name
		if inputs[i].Required != inputs[j].Required {
			return inputs[i].Required
		}
		return inputs[i].Name < inputs[j].Name
	})

	// Derive component name from filename (without extension)
	base := filepath.Base(path)
	name := base[:len(base)-len(filepath.Ext(base))]

	return ComponentData{
		Name:        name,
		Description: loadComponentDescription(name),
		Inputs:      inputs,
	}, nil
}

// ensureTemplate checks if the template file exists, creates it from the default if missing
func ensureTemplate(path string, defaultContent []byte) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.WriteFile(path, defaultContent, 0644)
			if err != nil {
				return false, fmt.Errorf("error creating default %s: %w", path, err)
			}
			return true, nil
		}
		return false, fmt.Errorf("error checking %s: %w", path, err)
	}
	return false, nil
}

func main() {
	projectPath := flag.String("project-path", "", "GitLab project path (e.g. group/project)")
	version := flag.String("version", "", "Component version (e.g. 1.0.0)")
	flag.Parse()

	// If README.md.tmpl doesn't exist, create it from the embedded default
	created, err := ensureTemplate("README.md.tmpl", defaultTemplate)
	if err != nil {
		fmt.Println(err)
		return
	}
	if created {
		fmt.Println("Created default README.md.tmpl")
	}

	// Find all templates in the templates/ directory
	templates, err := filepath.Glob("templates/*.yml")
	if err != nil {
		fmt.Printf("Error finding template files: %s\n", err)
		return
	}

	if len(templates) == 0 {
		fmt.Println("No template files found in templates/")
		return
	}

	sort.Strings(templates)

	// Parse all templates
	var components []ComponentData
	for _, t := range templates {
		component, err := parseTemplate(t)
		if err != nil {
			fmt.Printf("%s\n", err)
			return
		}
		components = append(components, component)
	}

	templateData := TemplateData{
		ProjectPath: resolveProjectPath(*projectPath),
		Version:     resolveVersion(*version),
		Components:  components,
	}

	// Read the template file
	tmpl, err := template.ParseFiles("README.md.tmpl")
	if err != nil {
		fmt.Printf("Error reading template file: %s\n", err)
		return
	}

	// Execute the template with data
	var doc bytes.Buffer
	err = tmpl.Execute(&doc, templateData)
	if err != nil {
		fmt.Printf("Error executing template: %s\n", err)
		return
	}

	// Write the documentation file
	err = os.WriteFile("README.md", doc.Bytes(), 0644)
	if err != nil {
		fmt.Printf("Error writing Markdown file: %s\n", err)
		return
	}

	fmt.Println("Documentation generated successfully!")
}
