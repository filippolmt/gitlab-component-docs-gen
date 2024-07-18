package main

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"github.com/goccy/go-yaml"
)

// Struct per rappresentare gli input del YAML
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

// Struct per rappresentare i dati per il template
type InputData struct {
	Name        string
	Description string
	Required    bool
	Default     string
}

type TemplateData struct {
	Inputs []InputData
}

func main() {
	// Leggi il file YAML
	yamlFile, err := os.ReadFile("templates/base.yml")
	if err != nil {
		fmt.Printf("Error reading YAML file: %s\n", err)
		return
	}

	// Decodifica il file YAML
	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
		return
	}

	// Prepara i dati per il template
	var inputData []InputData
	for name, input := range config.Spec.Inputs {
		inputData = append(inputData, InputData{
			Name:        name,
			Description: input.Description,
			Required:    input.Default == "",
			Default:     input.Default,
		})
	}

	templateData := TemplateData{
		Inputs: inputData,
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
