package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"dtoForge/internal/generator"
	"dtoForge/internal/typescript"
)

// Config holds the command-line configuration.
type Config struct {
	OpenAPIFile    string
	OutputFolder   string
	TargetLanguage string
	PackageName    string
	ConfigFile     string // New: path to config file
}

// OpenAPISpec is a minimal representation of an OpenAPI 3 spec.
type OpenAPISpec struct {
	OpenAPI    string                 `yaml:"openapi"`
	Info       map[string]interface{} `yaml:"info"`
	Paths      map[string]interface{} `yaml:"paths"`
	Components map[string]interface{} `yaml:"components"`
}

func parseCLIArgs() Config {
	openAPIFile := flag.String("openapi", "", "Path to the OpenAPI spec file (JSON or YAML)")
	outputFolder := flag.String("out", "./generated", "Output folder for generated files")
	targetLang := flag.String("lang", "typescript", "Target language (typescript)")
	packageName := flag.String("package", "", "Package/module name (optional)")
	configFile := flag.String("config", "", "Path to dtoforge config file (optional)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "DtoForge - OpenAPI to io-ts code generator\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nConfig file discovery (if -config not specified):\n")
		fmt.Fprintf(os.Stderr, "  1. ./dtoforge.config.yaml (current directory)\n")
		fmt.Fprintf(os.Stderr, "  2. Same directory as OpenAPI file\n")
		fmt.Fprintf(os.Stderr, "  3. Same directory as binary\n")
		fmt.Fprintf(os.Stderr, "\nExample config file can be generated with: %s -example-config\n", os.Args[0])
	}

	// Special flag to generate example config
	exampleConfig := flag.Bool("example-config", false, "Generate example dtoforge.config.yaml and exit")

	flag.Parse()

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
	}
}

// discoverConfigFile finds the config file using the discovery logic
func discoverConfigFile(config Config) string {
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

// ... rest of the functions remain the same (readOpenAPISpec, convertToGeneratorDTOs, etc.)

func readOpenAPISpec(path string) (*OpenAPISpec, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	var spec OpenAPISpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	return &spec, nil
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
			for propName, propVal := range props {
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
		Name:     name,
		Metadata: make(map[string]string),
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

	// Handle enum within property
	if enumVals, ok := schema["enum"].([]interface{}); ok {
		var values []string
		underlyingType := "string"
		if typ, ok := schema["type"].(string); ok {
			underlyingType = typ
		}

		for _, val := range enumVals {
			if strVal, ok := val.(string); ok {
				values = append(values, strVal)
			}
		}

		prop.Type = generator.EnumType{
			Name:           fmt.Sprintf("%sEnum", strings.Title(name)),
			UnderlyingType: underlyingType,
			Values:         values,
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

	// Create generator registry
	registry := generator.NewRegistry()

	// Create TypeScript generator and pass the config file path
	tsGen := typescript.NewTypeScriptGenerator()
	registry.Register(tsGen)

	// Get the appropriate generator
	gen, err := registry.Get(config.TargetLanguage)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("Available languages: %v\n", registry.Available())
		os.Exit(1)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(config.OutputFolder, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Read and parse OpenAPI spec
	spec, err := readOpenAPISpec(config.OpenAPIFile)
	if err != nil {
		fmt.Printf("Error reading OpenAPI spec: %v\n", err)
		os.Exit(1)
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

	// Discover config file
	configFile := discoverConfigFile(config)
	if configFile != "" {
		fmt.Printf("📝 Using config file: %s\n", configFile)
	} else {
		fmt.Printf("📝 No config file found, using defaults\n")
	}

	// Generate code
	genConfig := generator.Config{
		OutputFolder:   config.OutputFolder,
		PackageName:    config.PackageName,
		TargetLanguage: config.TargetLanguage,
		ConfigFile:     configFile, // Pass the discovered config file path
	}

	if err := gen.Generate(dtos, genConfig); err != nil {
		fmt.Printf("Error generating code: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🚀 Successfully generated %s code in %s\n", config.TargetLanguage, config.OutputFolder)
}
