package main

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

// Struct representing YAML inputs
type Inputs struct {
	Description string `yaml:"description"`
	Default     string `yaml:"default"`
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
	Name   string
	Inputs []InputData
}

type TemplateData struct {
	Components []ComponentData
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
			Required:    input.Default == "",
			Default:     input.Default,
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
		Name:   name,
		Inputs: inputs,
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
		Components: components,
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
