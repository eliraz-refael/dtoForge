package typescript

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"
	"dtoForge/internal/generator"
)

// CustomTypeMapping defines how to map OpenAPI formats to TypeScript/io-ts types
type CustomTypeMapping struct {
	// The io-ts codec to use (e.g., "DateFromISOString", "UUID.codec")
	IoTsType string `yaml:"ioTsType"`
	// The TypeScript type (e.g., "Date", "UUID", "EmailString")
	TypeScriptType string `yaml:"typeScriptType"`
	// Import statement needed (e.g., "import { DateFromISOString } from 'io-ts-types'")
	ImportStatement string `yaml:"import"`
}

// CustomTypeConfig represents the YAML configuration structure
type CustomTypeConfig struct {
	CustomTypes map[string]CustomTypeMapping `yaml:"customTypes"`
}

// CustomTypeRegistry holds all custom type mappings
type CustomTypeRegistry struct {
	mappings map[string]CustomTypeMapping
}

// NewCustomTypeRegistry creates a new registry with default mappings
func NewCustomTypeRegistry() *CustomTypeRegistry {
	registry := &CustomTypeRegistry{
		mappings: make(map[string]CustomTypeMapping),
	}

	// Add default mappings
	registry.addDefaultMappings()

	return registry
}

// addDefaultMappings adds the built-in format mappings
func (r *CustomTypeRegistry) addDefaultMappings() {
	r.mappings["date-time"] = CustomTypeMapping{
		IoTsType:        "DateFromISOString",
		TypeScriptType:  "Date",
		ImportStatement: "import { DateFromISOString } from 'io-ts-types';",
	}

	// Add more common formats with sensible defaults
	r.mappings["uuid"] = CustomTypeMapping{
		IoTsType:        "t.string",
		TypeScriptType:  "string",
		ImportStatement: "",
	}

	r.mappings["email"] = CustomTypeMapping{
		IoTsType:        "t.string",
		TypeScriptType:  "string",
		ImportStatement: "",
	}

	r.mappings["uri"] = CustomTypeMapping{
		IoTsType:        "t.string",
		TypeScriptType:  "string",
		ImportStatement: "",
	}

	r.mappings["date"] = CustomTypeMapping{
		IoTsType:        "t.string",
		TypeScriptType:  "string",
		ImportStatement: "",
	}
}

// Register adds or updates a custom type mapping
func (r *CustomTypeRegistry) Register(format string, mapping CustomTypeMapping) {
	r.mappings[format] = mapping
}

// Get retrieves a mapping for a given format
func (r *CustomTypeRegistry) Get(format string) (CustomTypeMapping, bool) {
	mapping, exists := r.mappings[format]
	return mapping, exists
}

// GetAllImports returns all unique import statements needed for used formats
func (r *CustomTypeRegistry) GetAllImports(usedFormats []string) []string {
	importSet := make(map[string]bool)
	var imports []string

	// Always include io-ts
	imports = append(imports, "import * as t from 'io-ts';")

	for _, format := range usedFormats {
		if mapping, exists := r.mappings[format]; exists && mapping.ImportStatement != "" {
			if !importSet[mapping.ImportStatement] {
				imports = append(imports, mapping.ImportStatement)
				importSet[mapping.ImportStatement] = true
			}
		}
	}

	return imports
}

// LoadFromConfig loads custom mappings from a YAML configuration file
func (r *CustomTypeRegistry) LoadFromConfig(configPath string) error {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil // Config file is optional
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config CustomTypeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	// Register all custom types from config
	for format, mapping := range config.CustomTypes {
		r.Register(format, mapping)
	}

	return nil
}

// SaveExampleConfig creates an example configuration file
func (r *CustomTypeRegistry) SaveExampleConfig(configPath string) error {
	exampleConfig := CustomTypeConfig{
		CustomTypes: map[string]CustomTypeMapping{
			"date-time": {
				IoTsType:        "DateTimeString",
				TypeScriptType:  "DateTimeString",
				ImportStatement: "import { DateTimeString } from './branded-types';",
			},
			"uuid": {
				IoTsType:        "UUID.codec",
				TypeScriptType:  "UUID",
				ImportStatement: "import { UUID } from './branded-types';",
			},
			"email": {
				IoTsType:        "EmailString.codec",
				TypeScriptType:  "EmailString",
				ImportStatement: "import { EmailString } from './branded-types';",
			},
		},
	}

	data, err := yaml.Marshal(exampleConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal example config: %w", err)
	}

	if err := ioutil.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write example config: %w", err)
	}

	return nil
}

// GetUsedFormats analyzes DTOs to find all used formats
func (r *CustomTypeRegistry) GetUsedFormats(dtos []generator.DTO) []string {
	formatSet := make(map[string]bool)
	var formats []string

	for _, dto := range dtos {
		for _, prop := range dto.Properties {
			if format := r.extractFormat(prop.Type); format != "" {
				if !formatSet[format] {
					formats = append(formats, format)
					formatSet[format] = true
				}
			}
		}
	}

	return formats
}

// extractFormat extracts format from a type using the GetFormat method
func (r *CustomTypeRegistry) extractFormat(irType generator.IRType) string {
	// Check if it's a type that has a GetFormat method
	type formatProvider interface {
		GetFormat() string
	}

	if fp, ok := irType.(formatProvider); ok {
		return fp.GetFormat()
	}

	return ""
}
