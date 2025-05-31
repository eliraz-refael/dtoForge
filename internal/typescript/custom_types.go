package typescript

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// OutputConfig defines output behavior
type OutputConfig struct {
	Folder         string `yaml:"folder"`
	Mode           string `yaml:"mode"`           // "multiple" or "single"
	SingleFileName string `yaml:"singleFileName"` // for single file mode
}

// GenerationConfig defines what to generate
type GenerationConfig struct {
	GeneratePackageJson   bool `yaml:"generatePackageJson"`
	GeneratePartialCodecs bool `yaml:"generatePartialCodecs"`
	GenerateHelpers       bool `yaml:"generateHelpers"`
}

// CustomTypeMapping defines how to map OpenAPI formats to TypeScript/io-ts types
type CustomTypeMapping struct {
	IoTsType        string `yaml:"ioTsType"`
	TypeScriptType  string `yaml:"typeScriptType"`
	ImportStatement string `yaml:"import"`
}

// EnhancedCustomTypeConfig represents the complete YAML configuration structure
type EnhancedCustomTypeConfig struct {
	Output      OutputConfig                 `yaml:"output"`
	CustomTypes map[string]CustomTypeMapping `yaml:"customTypes"`
	Generation  GenerationConfig             `yaml:"generation"`
}

// CustomTypeRegistry holds all custom type mappings and config
type CustomTypeRegistry struct {
	mappings   map[string]CustomTypeMapping
	output     OutputConfig
	generation GenerationConfig
}

// NewCustomTypeRegistry creates a new registry with default mappings and config
func NewCustomTypeRegistry() *CustomTypeRegistry {
	registry := &CustomTypeRegistry{
		mappings: make(map[string]CustomTypeMapping),
		output: OutputConfig{
			Folder:         "./generated",
			Mode:           "multiple",
			SingleFileName: "schemas.ts",
		},
		generation: GenerationConfig{
			GeneratePackageJson:   true,
			GeneratePartialCodecs: true,
			GenerateHelpers:       true,
		},
	}

	registry.addDefaultMappings()
	return registry
}

// GetOutputConfig returns the output configuration
func (r *CustomTypeRegistry) GetOutputConfig() OutputConfig {
	return r.output
}

// GetGenerationConfig returns the generation configuration
func (r *CustomTypeRegistry) GetGenerationConfig() GenerationConfig {
	return r.generation
}

// IsSingleFileMode returns true if single file output is configured
func (r *CustomTypeRegistry) IsSingleFileMode() bool {
	return r.output.Mode == "single"
}

// GetSingleFileName returns the filename for single file mode
func (r *CustomTypeRegistry) GetSingleFileName() string {
	if r.output.SingleFileName == "" {
		return "schemas.ts"
	}
	return r.output.SingleFileName
}

// addDefaultMappings adds the built-in format mappings
func (r *CustomTypeRegistry) addDefaultMappings() {
	r.mappings["date-time"] = CustomTypeMapping{
		IoTsType:        "DateFromISOString",
		TypeScriptType:  "Date",
		ImportStatement: "import { DateFromISOString } from 'io-ts-types';",
	}

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

	// Always include io-ts first
	imports = append(imports, "import * as t from 'io-ts';")

	// Collect all custom type imports
	var customImports []string
	for _, format := range usedFormats {
		if mapping, exists := r.mappings[format]; exists && mapping.ImportStatement != "" {
			if !importSet[mapping.ImportStatement] {
				customImports = append(customImports, mapping.ImportStatement)
				importSet[mapping.ImportStatement] = true
			}
		}
	}

	// Sort custom imports alphabetically for consistent output
	sort.Strings(customImports)
	imports = append(imports, customImports...)

	return imports
}

// LoadFromConfig loads custom mappings from a YAML configuration file
func (r *CustomTypeRegistry) LoadFromConfig(configPath string) error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil // Config file is optional
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config EnhancedCustomTypeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	// Load output config if provided
	if config.Output.Folder != "" || config.Output.Mode != "" || config.Output.SingleFileName != "" {
		if config.Output.Folder != "" {
			r.output.Folder = config.Output.Folder
		}
		if config.Output.Mode != "" {
			if config.Output.Mode != "multiple" && config.Output.Mode != "single" {
				return fmt.Errorf("invalid output mode '%s', must be 'multiple' or 'single'", config.Output.Mode)
			}
			r.output.Mode = config.Output.Mode
		}
		if config.Output.SingleFileName != "" {
			r.output.SingleFileName = config.Output.SingleFileName
		}
	}

	// Load generation config if provided
	r.generation.GeneratePackageJson = config.Generation.GeneratePackageJson
	r.generation.GeneratePartialCodecs = config.Generation.GeneratePartialCodecs
	r.generation.GenerateHelpers = config.Generation.GenerateHelpers

	// Register all custom types from config
	for format, mapping := range config.CustomTypes {
		r.Register(format, mapping)
	}

	return nil
}

// SaveExampleConfig creates an example configuration file
func (r *CustomTypeRegistry) SaveExampleConfig(configPath string) error {
	exampleConfig := EnhancedCustomTypeConfig{
		Output: OutputConfig{
			Folder:         "./generated",
			Mode:           "multiple",
			SingleFileName: "schemas.ts",
		},
		Generation: GenerationConfig{
			GeneratePackageJson:   true,
			GeneratePartialCodecs: true,
			GenerateHelpers:       true,
		},
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

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write example config: %w", err)
	}

	return nil
}
