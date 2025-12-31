package zod

import (
	"fmt"
	"io/ioutil"
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
	GeneratePackageJson    bool `yaml:"generatePackageJson"`
	GenerateHelpers        bool `yaml:"generateHelpers"`
	GenerateSchemaRegistry bool `yaml:"generateSchemaRegistry"`
	GenerateSchemaNames    bool `yaml:"generateSchemaNames"`
	ProcessImplicitObjects bool `yaml:"processImplicitObjects"` // Process schemas with properties but no explicit type
}

// CustomTypeMapping defines how to map OpenAPI formats to Zod types
type CustomTypeMapping struct {
	ZodType        string `yaml:"zodType"`
	TypeScriptType string `yaml:"typeScriptType"`
	Import         string `yaml:"import"`
}

// XNullableBehavior represents the behavior when x-nullable is true
type XNullableBehavior string

const (
	// XNullableBehaviorNull adds null to the type (.nullable())
	XNullableBehaviorNull XNullableBehavior = "null"
	// XNullableBehaviorUndefined makes the field optional (.optional())
	XNullableBehaviorUndefined XNullableBehavior = "undefined"
	// XNullableBehaviorNullish makes the field both nullable and optional
	XNullableBehaviorNullish XNullableBehavior = "nullish"
)

// XNullableConfig represents x-nullable configuration
type XNullableConfig struct {
	Enabled  *bool             `yaml:"enabled"`  // Pointer to detect if it was set
	Behavior XNullableBehavior `yaml:"behavior"` // "null", "undefined", or "nullish"
}

// ExtensionConfig represents configuration for OpenAPI extensions
type ExtensionConfig struct {
	XNullable XNullableConfig `yaml:"x-nullable"`
}

// ZodCustomTypeConfig represents the typescript-zod section in YAML configuration
type ZodCustomTypeConfig struct {
	Output      OutputConfig                 `yaml:"output"`
	CustomTypes map[string]CustomTypeMapping `yaml:"customTypes"`
	Generation  GenerationConfig             `yaml:"generation"`
	Extensions  ExtensionConfig              `yaml:"extensions"`
}

// FullConfig represents the complete YAML configuration structure
type FullConfig struct {
	TypeScriptZod ZodCustomTypeConfig `yaml:"typescript-zod"`
}

// CustomTypeRegistry holds all custom type mappings and config for Zod
type CustomTypeRegistry struct {
	mappings   map[string]CustomTypeMapping
	output     OutputConfig
	generation GenerationConfig
	extensions ExtensionConfig
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
			GeneratePackageJson:    true,
			GenerateHelpers:        true,
			GenerateSchemaRegistry: false,
			GenerateSchemaNames:    false,
			ProcessImplicitObjects: false, // Default to false for backward compatibility
		},
		extensions: ExtensionConfig{},
	}

	// Enable x-nullable by default
	enabled := true
	registry.extensions.XNullable.Enabled = &enabled

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

// IsXNullableEnabled returns true if x-nullable extension handling is enabled
func (r *CustomTypeRegistry) IsXNullableEnabled() bool {
	// If not set (nil), default to true
	if r.extensions.XNullable.Enabled == nil {
		return true
	}
	return *r.extensions.XNullable.Enabled
}

// GetXNullableBehavior returns the configured x-nullable behavior
func (r *CustomTypeRegistry) GetXNullableBehavior() XNullableBehavior {
	if r.extensions.XNullable.Behavior == "" {
		return XNullableBehaviorNull // Default to "null" for backward compatibility
	}
	return r.extensions.XNullable.Behavior
}

// addDefaultMappings adds the built-in format mappings for Zod
func (r *CustomTypeRegistry) addDefaultMappings() {
	r.mappings["date-time"] = CustomTypeMapping{
		ZodType:        "z.string().datetime()",
		TypeScriptType: "string",
		Import:         "",
	}

	r.mappings["uuid"] = CustomTypeMapping{
		ZodType:        "z.string().uuid()",
		TypeScriptType: "string",
		Import:         "",
	}

	r.mappings["email"] = CustomTypeMapping{
		ZodType:        "z.string().email()",
		TypeScriptType: "string",
		Import:         "",
	}

	r.mappings["uri"] = CustomTypeMapping{
		ZodType:        "z.string().url()",
		TypeScriptType: "string",
		Import:         "",
	}

	r.mappings["url"] = CustomTypeMapping{
		ZodType:        "z.string().url()",
		TypeScriptType: "string",
		Import:         "",
	}

	r.mappings["date"] = CustomTypeMapping{
		ZodType:        "z.string().date()",
		TypeScriptType: "string",
		Import:         "",
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

	// Always include Zod first
	imports = append(imports, "import { z } from 'zod';")

	// Collect all custom type imports
	var customImports []string
	for _, format := range usedFormats {
		if mapping, exists := r.mappings[format]; exists && mapping.Import != "" {
			if !importSet[mapping.Import] {
				customImports = append(customImports, mapping.Import)
				importSet[mapping.Import] = true
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

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config FullConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	zodConfig := config.TypeScriptZod

	// Load output config if provided
	if zodConfig.Output.Folder != "" || zodConfig.Output.Mode != "" || zodConfig.Output.SingleFileName != "" {
		if zodConfig.Output.Folder != "" {
			r.output.Folder = zodConfig.Output.Folder
		}
		if zodConfig.Output.Mode != "" {
			if zodConfig.Output.Mode != "multiple" && zodConfig.Output.Mode != "single" {
				return fmt.Errorf("invalid output mode '%s', must be 'multiple' or 'single'", zodConfig.Output.Mode)
			}
			r.output.Mode = zodConfig.Output.Mode
		}
		if zodConfig.Output.SingleFileName != "" {
			r.output.SingleFileName = zodConfig.Output.SingleFileName
		}
	}

	// Load generation config if provided
	r.generation.GeneratePackageJson = zodConfig.Generation.GeneratePackageJson
	r.generation.GenerateHelpers = zodConfig.Generation.GenerateHelpers
	r.generation.GenerateSchemaRegistry = zodConfig.Generation.GenerateSchemaRegistry
	r.generation.GenerateSchemaNames = zodConfig.Generation.GenerateSchemaNames

	// Load extensions config
	// Only update if explicitly set in config (pointer is not nil)
	if zodConfig.Extensions.XNullable.Enabled != nil {
		r.extensions.XNullable.Enabled = zodConfig.Extensions.XNullable.Enabled
	}
	// Load x-nullable behavior if set
	if zodConfig.Extensions.XNullable.Behavior != "" {
		r.extensions.XNullable.Behavior = zodConfig.Extensions.XNullable.Behavior
	}

	// Register all custom types from config
	for format, mapping := range zodConfig.CustomTypes {
		r.Register(format, mapping)
	}

	return nil
}

// SaveExampleConfig creates an example configuration file
func (r *CustomTypeRegistry) SaveExampleConfig(configPath string) error {
	exampleConfig := FullConfig{
		TypeScriptZod: ZodCustomTypeConfig{
			Output: OutputConfig{
				Folder:         "./generated",
				Mode:           "multiple",
				SingleFileName: "schemas.ts",
			},
			Generation: GenerationConfig{
				GeneratePackageJson:    true,
				GenerateHelpers:        true,
				GenerateSchemaRegistry: false,
				GenerateSchemaNames:    false,
			},
			CustomTypes: map[string]CustomTypeMapping{
				"date-time": {
					ZodType:        "DateTimeSchema",
					TypeScriptType: "DateTime",
					Import:         "import { DateTimeSchema } from './datetime-utils';",
				},
				"uuid": {
					ZodType:        "z.string().uuid().brand('UUID')",
					TypeScriptType: "UUID",
					Import:         "",
				},
				"email": {
					ZodType:        "EmailSchema",
					TypeScriptType: "Email",
					Import:         "import { EmailSchema } from './branded-types';",
				},
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
