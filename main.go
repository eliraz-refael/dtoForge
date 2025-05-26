package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the command-line configuration.
type Config struct {
	OpenAPIFile  string // Path to the OpenAPI spec file (JSON or YAML)
	OutputFolder string // Folder where generated DTO files will be written
}

// DTO represents a Data Transfer Object in our IR.
type DTO struct {
	Name        string     // e.g., "Item", "NewItem"
	Description string     // Optional description from the spec
	Properties  []Property // List of properties for the DTO
	Required    []string   // List of required property names
}

// Property represents a field within a DTO.
type Property struct {
	Name          string  // Field name (e.g., "id", "name")
	Type          IRType  // The type of the property
	Description   string  // Optional documentation for the property
	Nullable      bool    // Whether the property can be null
	CustomBranded string  // Optional override for a custom/branded type (e.g., "DateTimeString")
}

// IRType is an interface for our type representations.
type IRType interface{}

// PrimitiveType represents basic types like string, number, etc.
type PrimitiveType struct {
	TypeName string // e.g., "string", "number", "boolean"
}

// ObjectType represents a nested object type.
type ObjectType struct {
	DTORef *DTO // Reference to another DTO (or inline DTO)
}

// ArrayType represents an array of elements.
type ArrayType struct {
	ElementType IRType // Type of the elements in the array
}

// ReferenceType represents a reference to an already defined DTO.
type ReferenceType struct {
	RefName string // Name of the referenced DTO
}

// EnumType represents an enum type.
type EnumType struct {
	UnderlyingType string   // e.g., "string"
	Values         []string // Allowed values
}

// OpenAPISpec is a minimal representation of an OpenAPI 3 spec.
// In a complete solution, you might generate a fully typed version of the spec.
type OpenAPISpec struct {
	OpenAPI    string                 `yaml:"openapi"`
	Info       map[string]interface{} `yaml:"info"`
	Paths      map[string]interface{} `yaml:"paths"`
	Components map[string]interface{} `yaml:"components"`
}

// parseCLIArgs parses the command-line arguments into a Config.
func parseCLIArgs() Config {
	openAPIFile := flag.String("openapi", "", "Path to the OpenAPI spec file (JSON or YAML)")
	outputFolder := flag.String("out", "./out", "Output folder for generated DTO files")
	flag.Parse()

	if *openAPIFile == "" {
		fmt.Println("Error: OpenAPI spec file is required. Use the -openapi flag.")
		os.Exit(1)
	}

	return Config{
		OpenAPIFile:  *openAPIFile,
		OutputFolder: *outputFolder,
	}
}

// readOpenAPISpec reads the file from the given path and parses it into an OpenAPISpec.
func readOpenAPISpec(path string) (*OpenAPISpec, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var spec OpenAPISpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, err
	}

	return &spec, nil
}

// convertToIR converts a parsed OpenAPISpec into our intermediate representation (IR).
func convertToIR(spec *OpenAPISpec) ([]DTO, error) {
	var dtos []DTO

	// Look for schemas under components.
	if comp, ok := spec.Components["schemas"]; ok {
		if schemas, ok := comp.(map[string]interface{}); ok {
			for name, schemaVal := range schemas {
				if schema, ok := schemaVal.(map[string]interface{}); ok {
					dto := convertSchemaToDTO(name, schema)
					dtos = append(dtos, dto)
				}
			}
		}
	}

	// You can later extend this to also extract inline schemas from paths.
	return dtos, nil
}

// convertSchemaToDTO converts a single schema (from components.schemas) into a DTO.
func convertSchemaToDTO(name string, schema map[string]interface{}) DTO {
	dto := DTO{
		Name:       name,
		Properties: []Property{},
		Required:   []string{},
	}

	// Optional description.
	if desc, ok := schema["description"].(string); ok {
		dto.Description = desc
	}

	// Capture required fields if present.
	if req, ok := schema["required"].([]interface{}); ok {
		for _, r := range req {
			if s, ok := r.(string); ok {
				dto.Required = append(dto.Required, s)
			}
		}
	}

	// Process object properties.
	if typ, ok := schema["type"].(string); ok && typ == "object" {
		if props, ok := schema["properties"].(map[string]interface{}); ok {
			for propName, propVal := range props {
				if propSchema, ok := propVal.(map[string]interface{}); ok {
					property := convertSchemaToProperty(propName, propSchema)
					dto.Properties = append(dto.Properties, property)
				}
			}
		}
	}

	return dto
}

// convertSchemaToProperty converts a property schema into a Property IR.
func convertSchemaToProperty(name string, schema map[string]interface{}) Property {
	prop := Property{
		Name: name,
	}

	if desc, ok := schema["description"].(string); ok {
		prop.Description = desc
	}

	// Check if the property is nullable.
	if nullable, ok := schema["nullable"].(bool); ok {
		prop.Nullable = nullable
	}

	// Determine the type of the property.
	if typ, ok := schema["type"].(string); ok {
		switch typ {
		case "string", "number", "boolean", "integer":
			prop.Type = PrimitiveType{TypeName: typ}
		case "array":
			// For arrays, get the items field.
			if items, ok := schema["items"].(map[string]interface{}); ok {
				itemProp := convertSchemaToProperty(name, items)
				prop.Type = ArrayType{ElementType: itemProp.Type}
			}
		case "object":
			// For nested objects, check for a $ref or process inline.
			if ref, ok := schema["$ref"].(string); ok {
				refName := extractRefName(ref)
				prop.Type = ReferenceType{RefName: refName}
			} else {
				// Inline object: treat it as an inline DTO.
				nestedDTO := convertSchemaToDTO(name, schema)
				prop.Type = ObjectType{DTORef: &nestedDTO}
			}
		default:
			// Fallback to a primitive.
			prop.Type = PrimitiveType{TypeName: typ}
		}
	} else if ref, ok := schema["$ref"].(string); ok {
		// If there's no type but a $ref exists, it's a reference.
		refName := extractRefName(ref)
		prop.Type = ReferenceType{RefName: refName}
	}

	return prop
}

// extractRefName extracts the schema name from a $ref string (e.g., "#/components/schemas/Item").
func extractRefName(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}

func main() {
	// Step 1: Parse CLI arguments.
	config := parseCLIArgs()

	// Step 2: Read and parse the OpenAPI spec.
	spec, err := readOpenAPISpec(config.OpenAPIFile)
	if err != nil {
		fmt.Printf("Error reading OpenAPI spec: %v\n", err)
		os.Exit(1)
	}

	// Step 3: Convert the spec into our intermediate representation (IR).
	ir, err := convertToIR(spec)
	if err != nil {
		fmt.Printf("Error converting spec to IR: %v\n", err)
		os.Exit(1)
	}

	// For now, we simply print the generated IR.
	fmt.Printf("Generated IR: %+v\n", ir)
}
