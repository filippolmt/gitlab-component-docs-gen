package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/template"

	"github.com/goccy/go-yaml"
)

// Struct per rappresentare gli input del YAML
type Inputs struct {
	Description string `yaml:"description"`
	Default     string `yaml:"default"`
}

type Spec struct {
	Description string            `yaml:"description"`
	Inputs      map[string]Inputs `yaml:"inputs"`
}

type Config struct {
	Spec Spec `yaml:"spec"`
}

// Struct per rappresentare i dati per il template
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
		// Metti i required prima, poi ordina per nome
		if inputs[i].Required != inputs[j].Required {
			return inputs[i].Required
		}
		return inputs[i].Name < inputs[j].Name
	})

	// Ricava il nome del componente dal nome del file (senza estensione)
	base := filepath.Base(path)
	name := base[:len(base)-len(filepath.Ext(base))]

	return ComponentData{
		Name:        name,
		Description: config.Spec.Description,
		Inputs:      inputs,
	}, nil
}

func main() {
	// Cerca tutti i template nella directory templates/
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

	// Parsa tutti i template
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

	// Leggi il file di template
	tmpl, err := template.ParseFiles("README.md.tmpl")
	if err != nil {
		fmt.Printf("Error reading template file: %s\n", err)
		return
	}

	// Esegui il template con i dati
	var doc bytes.Buffer
	err = tmpl.Execute(&doc, templateData)
	if err != nil {
		fmt.Printf("Error executing template: %s\n", err)
		return
	}

	// Scrivi il file di documentazione
	err = os.WriteFile("README.md", doc.Bytes(), 0644)
	if err != nil {
		fmt.Printf("Error writing Markdown file: %s\n", err)
		return
	}

	fmt.Println("Documentation generated successfully!")
}
