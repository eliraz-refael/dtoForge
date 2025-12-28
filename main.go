package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"dtoForge/internal/generator"
	"dtoForge/internal/typescript"
	"dtoForge/internal/zod"
)

// Version is set at build time via ldflags
// e.g., go build -ldflags "-X main.Version=v1.4.7"
var Version = "dev"

type Config struct {
	OpenAPIFile    string
	OutputFolder   string
	TargetLanguage string
	PackageName    string
	ConfigFile     string
	NoConfig       bool
}

type OpenAPISpec struct {
	OpenAPI    string                 `yaml:"openapi"`
	Info       map[string]interface{} `yaml:"info"`
	Paths      map[string]interface{} `yaml:"paths"`
	Components map[string]interface{} `yaml:"components"`
}

func parseCLIArgs() Config {
	openAPIFile := flag.String("openapi", "", "Path to the OpenAPI spec file (JSON or YAML)")
	outputFolder := flag.String("out", "./generated", "Output folder for generated files")
	targetLang := flag.String("lang", "typescript", "Target language (typescript, typescript-zod)")
	packageName := flag.String("package", "", "Package/module name (optional)")
	configFile := flag.String("config", "", "Path to dtoforge config file (optional)")
	noConfig := flag.Bool("no-config", false, "Disable automatic config file discovery")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "DtoForge %s - OpenAPI to TypeScript schema generator\n\n", Version)
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nSupported languages:\n")
		fmt.Fprintf(os.Stderr, "  typescript     - TypeScript with io-ts validation (default)\n")
		fmt.Fprintf(os.Stderr, "  typescript-zod - TypeScript with Zod validation\n")
		fmt.Fprintf(os.Stderr, "\nConfig file discovery (if -config not specified and -no-config not set):\n")
		fmt.Fprintf(os.Stderr, "  1. ./dtoforge.config.yaml (current directory)\n")
		fmt.Fprintf(os.Stderr, "  2. Same directory as OpenAPI file\n")
		fmt.Fprintf(os.Stderr, "  3. Same directory as binary\n")
		fmt.Fprintf(os.Stderr, "\nExample config file can be generated with: %s -example-config\n", os.Args[0])
	}

	// Special flags
	showVersion := flag.Bool("version", false, "Print version and exit")
	exampleConfig := flag.Bool("example-config", false, "Generate example dtoforge.config.yaml and exit")

	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("DtoForge %s\n", Version)
		os.Exit(0)
	}

	// Handle example config generation
	if *exampleConfig {
		if err := generateExampleConfig(); err != nil {
			fmt.Printf("Error generating example config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Generated dtoforge.config.yaml example file")
		os.Exit(0)
	}

	if *openAPIFile == "" {
		fmt.Println("Error: OpenAPI spec file is required. Use the -openapi flag.")
		flag.Usage()
		os.Exit(1)
	}

	return Config{
		OpenAPIFile:    *openAPIFile,
		OutputFolder:   *outputFolder,
		TargetLanguage: *targetLang,
		PackageName:    *packageName,
		ConfigFile:     *configFile,
		NoConfig:       *noConfig,
	}
}

// discoverConfigFile finds the config file using the discovery logic
func discoverConfigFile(config Config) string {
	// If --no-config flag is set, return empty string (no config)
	if config.NoConfig {
		return ""
	}

	// If explicitly specified, use that
	if config.ConfigFile != "" {
		return config.ConfigFile
	}

	configName := "dtoforge.config.yaml"

	// 1. Current directory
	if _, err := os.Stat(configName); err == nil {
		return configName
	}

	// 2. Same directory as OpenAPI file
	openAPIDir := filepath.Dir(config.OpenAPIFile)
	configPath := filepath.Join(openAPIDir, configName)
	if _, err := os.Stat(configPath); err == nil {
		return configPath
	}

	// 3. Same directory as binary
	if execPath, err := os.Executable(); err == nil {
		binaryDir := filepath.Dir(execPath)
		configPath := filepath.Join(binaryDir, configName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	// Return empty string if not found (will use defaults)
	return ""
}

// generateExampleConfig creates an example configuration file
func generateExampleConfig() error {
	registry := typescript.NewCustomTypeRegistry()
	return registry.SaveExampleConfig("dtoforge.config.yaml")
}

// filterDTOs applies exclusion rules to DTOs based on config
func filterDTOs(dtos []generator.DTO, excludeConfig *typescript.ExcludeConfig) []generator.DTO {
	if excludeConfig == nil {
		return dtos
	}

	var filtered []generator.DTO
	var excluded []string

	for _, dto := range dtos {
		if excludeConfig.ShouldExclude(dto.Name) {
			excluded = append(excluded, dto.Name)
		} else {
			filtered = append(filtered, dto)
		}
	}

	if len(excluded) > 0 {
		fmt.Printf("📝 Excluded %d schemas: %v\n", len(excluded), excluded)
	}

	return filtered
}

// ... rest of the functions remain the same (readOpenAPISpec, convertToGeneratorDTOs, etc.)

func readOpenAPISpec(path string) (*OpenAPISpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	var spec OpenAPISpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	return &spec, nil
}

// preprocessSchemas modifies the OpenAPI spec to add explicit type declarations
// for schemas that have properties but no explicit type
func preprocessSchemas(spec *OpenAPISpec, processImplicitObjects bool) {
	if !processImplicitObjects {
		return
	}

	if comp, ok := spec.Components["schemas"]; ok {
		if schemas, ok := comp.(map[string]interface{}); ok {
			for name, schemaVal := range schemas {
				if schema, ok := schemaVal.(map[string]interface{}); ok {
					// If has properties but no explicit type, add type: object
					if _, hasProperties := schema["properties"]; hasProperties {
						if _, hasType := schema["type"]; !hasType {
							schema["type"] = "object"
							fmt.Printf("ℹ️  Auto-added type 'object' to schema '%s' (has properties but no explicit type)\n", name)
						}
					}
				}
			}
		}
	}
}

func convertToGeneratorDTOs(spec *OpenAPISpec) ([]generator.DTO, error) {
	var dtos []generator.DTO

	if comp, ok := spec.Components["schemas"]; ok {
		if schemas, ok := comp.(map[string]interface{}); ok {
			for name, schemaVal := range schemas {
				if schema, ok := schemaVal.(map[string]interface{}); ok {
					dto, err := convertSchemaToGeneratorDTO(name, schema)
					if err != nil {
						return nil, fmt.Errorf("failed to convert schema %s: %w", name, err)
					}
					dtos = append(dtos, dto)
				}
			}
		}
	}

	return dtos, nil
}

// parseAllOf extracts types from an allOf array in OpenAPI schemas.
// It handles both $ref references to other schemas and inline schema definitions.
// Returns an error if parsing fails or if allOf contains fewer than 2 types (semantic validation).
func parseAllOf(allOf []interface{}, namePrefix string) ([]generator.IRType, error) {
	var types []generator.IRType

	for i, item := range allOf {
		itemSchema, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("allOf item %d is not a valid schema object", i)
		}

		// Check if it's a reference to another schema
		if ref, ok := itemSchema["$ref"].(string); ok {
			refName := extractRefName(ref)
			types = append(types, generator.ReferenceType{RefName: refName})
		} else {
			// It's an inline schema - convert it to extract the type
			tempProp, err := convertSchemaToGeneratorProperty(fmt.Sprintf("%s_allOfItem%d", namePrefix, i), itemSchema, []string{})
			if err != nil {
				return nil, fmt.Errorf("failed to convert allOf item %d: %w", i, err)
			}
			types = append(types, tempProp.Type)
		}
	}

	// Validate that allOf contains at least 1 type
	// Note: Single-element allOf is technically valid but semantically redundant
	// We still support it as it appears in some OpenAPI specs
	if len(types) == 0 {
		return nil, fmt.Errorf("allOf must contain at least 1 schema, got 0")
	}

	return types, nil
}

func convertSchemaToGeneratorDTO(name string, schema map[string]interface{}) (generator.DTO, error) {
	dto := generator.DTO{
		Name:       name,
		Properties: []generator.Property{},
		Required:   []string{},
		Metadata:   make(map[string]string),
	}

	if desc, ok := schema["description"].(string); ok {
		dto.Description = desc
	}

	// Handle allOf at schema level (schema composition)
	// allOf represents intersection types where a schema is composed of multiple schemas
	if allOf, ok := schema["allOf"].([]interface{}); ok {
		types, err := parseAllOf(allOf, name)
		if err != nil {
			return dto, fmt.Errorf("failed to parse allOf for schema %s: %w", name, err)
		}

		// Store the intersection type in the dedicated field
		dto.Type = "allOf"
		intersectionType := generator.IntersectionType{Types: types}
		dto.IntersectionType = &intersectionType

		return dto, nil
	}

	// Handle enum types
	if enumVals, ok := schema["enum"].([]interface{}); ok {
		dto.Type = "enum"
		for _, val := range enumVals {
			if strVal, ok := val.(string); ok {
				dto.EnumValues = append(dto.EnumValues, strVal)
			}
		}
		return dto, nil
	}

	// Capture required fields
	if req, ok := schema["required"].([]interface{}); ok {
		for _, r := range req {
			if s, ok := r.(string); ok {
				dto.Required = append(dto.Required, s)
			}
		}
	}

	// Process object properties
	if typ, ok := schema["type"].(string); ok && typ == "object" {
		dto.Type = "object"
		if props, ok := schema["properties"].(map[string]interface{}); ok {
			// IMPORTANT: Sort property names for consistent ordering
			var propNames []string
			for propName := range props {
				propNames = append(propNames, propName)
			}
			sort.Strings(propNames) // This ensures consistent property order

			// Process properties in sorted order
			for _, propName := range propNames {
				propVal := props[propName]
				if propSchema, ok := propVal.(map[string]interface{}); ok {
					property, err := convertSchemaToGeneratorProperty(propName, propSchema, dto.Required)
					if err != nil {
						return dto, fmt.Errorf("failed to convert property %s: %w", propName, err)
					}
					dto.Properties = append(dto.Properties, property)
				}
			}
		}
	}

	return dto, nil
}

func convertSchemaToGeneratorProperty(name string, schema map[string]interface{}, required []string) (generator.Property, error) {
	prop := generator.Property{
		Name:       name,
		Metadata:   make(map[string]string),
		Extensions: make(map[string]interface{}),
	}

	// Check if property is required
	for _, req := range required {
		if req == name {
			prop.Required = true
			break
		}
	}

	if desc, ok := schema["description"].(string); ok {
		prop.Description = desc
	}

	if nullable, ok := schema["nullable"].(bool); ok {
		prop.Nullable = nullable
	}

	// Parse OpenAPI extensions (x-* fields)
	for key, value := range schema {
		if strings.HasPrefix(key, "x-") {
			prop.Extensions[key] = value
		}
	}

	// Handle enum within property
	if enumVals, ok := schema["enum"].([]interface{}); ok {
		var values []string
		hasNull := false
		underlyingType := "string"
		if typ, ok := schema["type"].(string); ok {
			underlyingType = typ
		}

		for _, val := range enumVals {
			if val == nil {
				hasNull = true
			} else if strVal, ok := val.(string); ok {
				values = append(values, strVal)
			}
		}

		prop.Type = generator.EnumType{
			Name:           fmt.Sprintf("%sEnum", strings.Title(name)),
			UnderlyingType: underlyingType,
			Values:         values,
			ContainsNull:   hasNull,
		}
		return prop, nil
	}

	// Determine the type of the property
	if typ, ok := schema["type"].(string); ok {
		switch typ {
		case "string":
			format := ""
			if f, ok := schema["format"].(string); ok {
				format = f
			}
			prop.Type = generator.PrimitiveType{Name: "string", Format: format}
		case "number", "integer":
			prop.Type = generator.PrimitiveType{Name: typ}
		case "boolean":
			prop.Type = generator.PrimitiveType{Name: "boolean"}
		case "array":
			if items, ok := schema["items"].(map[string]interface{}); ok {
				itemProp, err := convertSchemaToGeneratorProperty(name+"Item", items, []string{})
				if err != nil {
					return prop, err
				}
				prop.Type = generator.ArrayType{ElementType: itemProp.Type}
			}
		case "object":
			if ref, ok := schema["$ref"].(string); ok {
				refName := extractRefName(ref)
				prop.Type = generator.ReferenceType{RefName: refName}
			} else if additionalProps, ok := schema["additionalProperties"].(map[string]interface{}); ok {
				// Object with additionalProperties - it's a map/record
				valueProp, err := convertSchemaToGeneratorProperty(name+"Value", additionalProps, []string{})
				if err != nil {
					return prop, err
				}
				prop.Type = generator.MapType{ValueType: valueProp.Type}
			} else {
				// Inline object - create a nested DTO
				nestedDTO, err := convertSchemaToGeneratorDTO(name, schema)
				if err != nil {
					return prop, err
				}
				prop.Type = generator.ObjectType{DTORef: &nestedDTO, Inline: true}
			}
		default:
			prop.Type = generator.PrimitiveType{Name: typ}
		}
	} else if ref, ok := schema["$ref"].(string); ok {
		refName := extractRefName(ref)
		prop.Type = generator.ReferenceType{RefName: refName}
	} else if allOf, ok := schema["allOf"].([]interface{}); ok {
		// Handle allOf (intersection type / schema composition)
		types, err := parseAllOf(allOf, name)
		if err != nil {
			return prop, fmt.Errorf("failed to parse allOf for property %s: %w", name, err)
		}
		prop.Type = generator.IntersectionType{Types: types}
	} else if oneOf, ok := schema["oneOf"].([]interface{}); ok {
		// Handle oneOf (union type)
		var types []generator.IRType
		for _, item := range oneOf {
			if itemSchema, ok := item.(map[string]interface{}); ok {
				// Check if it's a reference
				if ref, ok := itemSchema["$ref"].(string); ok {
					refName := extractRefName(ref)
					types = append(types, generator.ReferenceType{RefName: refName})
				} else if typ, ok := itemSchema["type"].(string); ok && typ == "null" {
					// Handle null type in oneOf
					types = append(types, generator.NullType{})
				} else {
					// Try to parse as inline schema
					itemProp, err := convertSchemaToGeneratorProperty(name+"Item", itemSchema, []string{})
					if err == nil {
						types = append(types, itemProp.Type)
					}
				}
			}
		}
		if len(types) > 0 {
			prop.Type = generator.UnionType{Types: types}
		} else {
			prop.Type = generator.PrimitiveType{Name: "unknown"}
		}
	} else {
		prop.Type = generator.PrimitiveType{Name: "unknown"}
	}

	return prop, nil
}

func extractRefName(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}

func main() {
	config := parseCLIArgs()

	registry := generator.NewRegistry()

	tsGen := typescript.NewTypeScriptGenerator()
	registry.Register(tsGen)

	zodGen := zod.NewZodGenerator()
	registry.Register(zodGen)

	// Get the appropriate generator
	gen, err := registry.Get(config.TargetLanguage)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("Available languages: %v\n", registry.Available())
		os.Exit(1)
	}

	// Discover config file BEFORE setting up output directory
	configFile := discoverConfigFile(config)
	if config.NoConfig {
		fmt.Printf("📝 Config file discovery disabled (--no-config flag)\n")
	} else if configFile != "" {
		fmt.Printf("📝 Using config file: %s\n", configFile)
	} else {
		fmt.Printf("📝 No config file found, using defaults\n")
	}

	// Load config to get default output folder if CLI didn't specify one
	finalOutputFolder := config.OutputFolder
	if configFile != "" {
		// Create a temporary registry just to load the config and get output settings
		tempRegistry := typescript.NewCustomTypeRegistry()
		if err := tempRegistry.LoadFromConfig(configFile); err != nil {
			fmt.Printf("Warning: Failed to load config file %s: %v\n", configFile, err)
		} else {
			outputConfig := tempRegistry.GetOutputConfig()
			// Only use config's output folder if CLI didn't specify one (still using default)
			if config.OutputFolder == "./generated" && outputConfig.Folder != "" {
				finalOutputFolder = outputConfig.Folder
				fmt.Printf("📁 Using output folder from config: %s\n", finalOutputFolder)
			}
		}
	}

	if err := os.MkdirAll(finalOutputFolder, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Read and parse OpenAPI spec
	spec, err := readOpenAPISpec(config.OpenAPIFile)
	if err != nil {
		fmt.Printf("Error reading OpenAPI spec: %v\n", err)
		os.Exit(1)
	}

	// Preprocess schemas if needed (add explicit type declarations for implicit objects)
	if configFile != "" {
		// Create a temporary registry just to load the config and get generation settings
		tempRegistry := typescript.NewCustomTypeRegistry()
		if err := tempRegistry.LoadFromConfig(configFile); err != nil {
			fmt.Printf("Warning: Failed to load config file %s for generation settings: %v\n", configFile, err)
		} else {
			genConfig := tempRegistry.GetGenerationConfig()
			if genConfig.ProcessImplicitObjects {
				fmt.Printf("🔧 Processing schemas with implicit object types enabled\n")
				preprocessSchemas(spec, true)
			}
		}
	}

	// Convert to generator DTOs
	dtos, err := convertToGeneratorDTOs(spec)
	if err != nil {
		fmt.Printf("Error converting spec to DTOs: %v\n", err)
		os.Exit(1)
	}

	if len(dtos) == 0 {
		fmt.Println("No schemas found in the OpenAPI spec")
		os.Exit(1)
	}

	fmt.Printf("✅ Successfully parsed %d schemas from OpenAPI spec\n", len(dtos))

	// Apply exclusion filters if configured
	if configFile != "" {
		tempRegistry := typescript.NewCustomTypeRegistry()
		if err := tempRegistry.LoadFromConfig(configFile); err != nil {
			fmt.Printf("Warning: Failed to load config file %s for exclusion filters: %v\n", configFile, err)
		} else {
			excludeConfig := tempRegistry.GetExcludeConfig()
			dtos = filterDTOs(dtos, excludeConfig)

			if len(dtos) == 0 {
				fmt.Println("⚠️  Warning: All schemas were excluded by filters")
				os.Exit(0)
			}
		}
	}

	// Generate code
	genConfig := generator.Config{
		OutputFolder:   finalOutputFolder,
		PackageName:    config.PackageName,
		TargetLanguage: config.TargetLanguage,
		ConfigFile:     configFile, // This will be empty if --no-config is used
		Version:        Version,
	}

	if err := gen.Generate(dtos, genConfig); err != nil {
		fmt.Printf("Error generating code: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🚀 DtoForge %s: Successfully generated %s code in %s\n", Version, config.TargetLanguage, finalOutputFolder)
}
