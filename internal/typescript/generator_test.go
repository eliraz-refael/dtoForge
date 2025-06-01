package typescript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dtoForge/internal/generator"
	"dtoForge/internal/testutils"
)

func TestTypeScriptGenerator_Language(t *testing.T) {
	gen := NewTypeScriptGenerator()
	if got := gen.Language(); got != "typescript" {
		t.Errorf("Language() = %v, want %v", got, "typescript")
	}
}

func TestTypeScriptGenerator_FileExtension(t *testing.T) {
	gen := NewTypeScriptGenerator()
	if got := gen.FileExtension(); got != ".ts" {
		t.Errorf("FileExtension() = %v, want %v", got, ".ts")
	}
}

func TestTypeScriptGenerator_ToIoTsType(t *testing.T) {
	gen := NewTypeScriptGenerator()
	gen.customTypes = NewCustomTypeRegistry()

	tests := []struct {
		name     string
		irType   generator.IRType
		nullable bool
		expected string
	}{
		{
			name:     "Basic string",
			irType:   generator.PrimitiveType{Name: "string"},
			nullable: false,
			expected: "t.string",
		},
		{
			name:     "String with email format",
			irType:   generator.PrimitiveType{Name: "string", Format: "email"},
			nullable: false,
			expected: "t.string",
		},
		{
			name:     "String with uuid format (with default mapping)",
			irType:   generator.PrimitiveType{Name: "string", Format: "uuid"},
			nullable: false,
			expected: "t.string",
		},
		{
			name:     "Nullable string",
			irType:   generator.PrimitiveType{Name: "string"},
			nullable: true,
			expected: "t.union([t.string, t.null])",
		},
		{
			name:     "Number type",
			irType:   generator.PrimitiveType{Name: "number"},
			nullable: false,
			expected: "t.number",
		},
		{
			name:     "Integer type",
			irType:   generator.PrimitiveType{Name: "integer"},
			nullable: false,
			expected: "t.number",
		},
		{
			name:     "Boolean type",
			irType:   generator.PrimitiveType{Name: "boolean"},
			nullable: false,
			expected: "t.boolean",
		},
		{
			name:     "Array of strings",
			irType:   generator.ArrayType{ElementType: generator.PrimitiveType{Name: "string"}},
			nullable: false,
			expected: "t.array(t.string)",
		},
		{
			name:     "Reference type",
			irType:   generator.ReferenceType{RefName: "User"},
			nullable: false,
			expected: "UserCodec",
		},
		{
			name:     "Enum type",
			irType:   generator.EnumType{Values: []string{"active", "inactive"}},
			nullable: false,
			expected: "t.keyof({'active': null, 'inactive': null})",
		},
		{
			name:     "Object type with reference",
			irType:   generator.ObjectType{RefName: "Product"},
			nullable: false,
			expected: "ProductCodec",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gen.toIoTsType(tt.irType, tt.nullable)
			if got != tt.expected {
				t.Errorf("toIoTsType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTypeScriptGenerator_ToTSType(t *testing.T) {
	gen := NewTypeScriptGenerator()
	gen.customTypes = NewCustomTypeRegistry()

	tests := []struct {
		name     string
		irType   generator.IRType
		nullable bool
		expected string
	}{
		{
			name:     "Basic string",
			irType:   generator.PrimitiveType{Name: "string"},
			nullable: false,
			expected: "string",
		},
		{
			name:     "Nullable string",
			irType:   generator.PrimitiveType{Name: "string"},
			nullable: true,
			expected: "string | null",
		},
		{
			name:     "Number type",
			irType:   generator.PrimitiveType{Name: "number"},
			nullable: false,
			expected: "number",
		},
		{
			name:     "Boolean type",
			irType:   generator.PrimitiveType{Name: "boolean"},
			nullable: false,
			expected: "boolean",
		},
		{
			name:     "Array of strings",
			irType:   generator.ArrayType{ElementType: generator.PrimitiveType{Name: "string"}},
			nullable: false,
			expected: "string[]",
		},
		{
			name:     "Reference type",
			irType:   generator.ReferenceType{RefName: "User"},
			nullable: false,
			expected: "User",
		},
		{
			name:     "Enum type",
			irType:   generator.EnumType{Values: []string{"active", "inactive"}},
			nullable: false,
			expected: "'active' | 'inactive'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gen.toTSType(tt.irType, tt.nullable)
			if got != tt.expected {
				t.Errorf("toTSType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTypeScriptGenerator_UtilityFunctions(t *testing.T) {
	gen := NewTypeScriptGenerator()

	tests := []struct {
		name     string
		function func(string) string
		input    string
		expected string
	}{
		{"CamelCase", gen.toCamelCase, "UserName", "userName"},
		{"CamelCase empty", gen.toCamelCase, "", ""},
		{"PascalCase", gen.toPascalCase, "userName", "UserName"},
		{"PascalCase empty", gen.toPascalCase, "", ""},
		{"KebabCase", gen.toKebabCase, "UserName", "user-name"},
		{"KebabCase already lowercase", gen.toKebabCase, "username", "username"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.function(tt.input)
			if got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestTypeScriptGenerator_HasDescription(t *testing.T) {
	gen := NewTypeScriptGenerator()

	tests := []struct {
		name        string
		description string
		expected    bool
	}{
		{"With description", "User information", true},
		{"Empty string", "", false},
		{"Whitespace only", "   ", false},
		{"Whitespace with content", "  description  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gen.hasDescription(tt.description)
			if got != tt.expected {
				t.Errorf("hasDescription() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTypeScriptGenerator_IsRequired(t *testing.T) {
	gen := NewTypeScriptGenerator()

	tests := []struct {
		name     string
		propName string
		required []string
		expected bool
	}{
		{"Property is required", "id", []string{"id", "name"}, true},
		{"Property is not required", "age", []string{"id", "name"}, false},
		{"Empty required list", "id", []string{}, false},
		{"Property not in list", "unknown", []string{"id", "name"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gen.isRequired(tt.propName, tt.required)
			if got != tt.expected {
				t.Errorf("isRequired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTypeScriptGenerator_Generate_MultipleFiles(t *testing.T) {
	gen := NewTypeScriptGenerator()
	tempDir := testutils.TempDir(t)

	// Create test DTOs
	dtos := []generator.DTO{
		testutils.CreateTestDTO("User"),
		{
			Name:        "Status",
			Type:        "enum",
			Description: "User status",
			EnumValues:  []string{"active", "inactive", "pending"},
		},
	}

	config := generator.Config{
		OutputFolder:   tempDir,
		PackageName:    "test-typescript",
		TargetLanguage: "typescript",
		ConfigFile:     "", // No config file
	}

	err := gen.Generate(dtos, config)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Check that expected files were created
	expectedFiles := []string{"user.ts", "status.ts", "index.ts", "package.json"}
	for _, filename := range expectedFiles {
		testutils.AssertFileExists(t, filepath.Join(tempDir, filename))
	}

	// Check content of user.ts
	userFile := filepath.Join(tempDir, "user.ts")
	testutils.AssertFileContains(t, userFile, "export const UserCodec = t.type({")
	testutils.AssertFileContains(t, userFile, "export type User = t.TypeOf<typeof UserCodec>;")
	testutils.AssertFileContains(t, userFile, "import * as t from 'io-ts';")

	// Check content of status.ts (enum)
	statusFile := filepath.Join(tempDir, "status.ts")
	testutils.AssertFileContains(t, statusFile, "export const StatusCodec = t.keyof(StatusValues);")
	testutils.AssertFileContains(t, statusFile, "'active': null")
	testutils.AssertFileContains(t, statusFile, "'inactive': null")
	testutils.AssertFileContains(t, statusFile, "'pending': null")

	// Check index.ts
	indexFile := filepath.Join(tempDir, "index.ts")
	testutils.AssertFileContains(t, indexFile, "export * from './user';")
	testutils.AssertFileContains(t, indexFile, "export * from './status';")
	testutils.AssertFileContains(t, indexFile, "export * as t from 'io-ts';")

	// Check package.json
	packageFile := filepath.Join(tempDir, "package.json")
	testutils.AssertFileContains(t, packageFile, `"io-ts": "^2.2.20"`)
	testutils.AssertFileContains(t, packageFile, `"name": "test-typescript"`)
}

func TestTypeScriptGenerator_Generate_SingleFile(t *testing.T) {
	gen := NewTypeScriptGenerator()
	tempDir := testutils.TempDir(t)

	// Create a config file for single file mode - use the correct structure
	configContent := `output:
  mode: single
  singleFileName: schemas.ts
generation:
  generatePackageJson: false
  generateHelpers: true
  generatePartialCodecs: true`

	configPath := testutils.WriteFile(t, tempDir, "config.yaml", configContent)

	dtos := []generator.DTO{
		testutils.CreateTestDTO("User"),
		{
			Name:        "Status",
			Type:        "enum",
			EnumValues:  []string{"active", "inactive"},
			Description: "Status enum",
		},
	}

	config := generator.Config{
		OutputFolder:   tempDir,
		PackageName:    "single-file-test",
		TargetLanguage: "typescript",
		ConfigFile:     configPath,
	}

	err := gen.Generate(dtos, config)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Should only have schemas.ts (no package.json due to config)
	testutils.AssertFileExists(t, filepath.Join(tempDir, "schemas.ts"))

	// Should NOT have individual files
	userFile := filepath.Join(tempDir, "user.ts")
	if _, err := os.Stat(userFile); err == nil {
		t.Error("Individual user.ts file should not exist in single file mode")
	}

	// Check single file content
	schemaFile := filepath.Join(tempDir, "schemas.ts")
	content := testutils.ReadFile(t, schemaFile)

	// Should contain both schemas
	if !strings.Contains(content, "export const UserCodec") {
		t.Error("Single file should contain UserCodec")
	}
	if !strings.Contains(content, "export const StatusCodec") {
		t.Error("Single file should contain StatusCodec")
	}

	// Should contain helper functions
	if !strings.Contains(content, "export const validateData") {
		t.Error("Single file should contain validateData helper")
	}

	// Should contain partial codecs
	if !strings.Contains(content, "UserPartialCodec") {
		t.Error("Single file should contain UserPartialCodec")
	}
}

func TestTypeScriptGenerator_CustomTypes(t *testing.T) {
	gen := NewTypeScriptGenerator()
	tempDir := testutils.TempDir(t)

	// Create config with custom types - use the correct structure
	configContent := `customTypes:
  uuid:
    ioTsType: "UUID"
    typeScriptType: "UUID"
    import: "import { UUID } from './custom-types';"
  email:
    ioTsType: "EmailString"
    typeScriptType: "EmailString"
    import: "import { EmailString } from './email-utils';"`

	configPath := testutils.WriteFile(t, tempDir, "config.yaml", configContent)

	// Create DTO with custom formats
	dto := generator.DTO{
		Name:        "CustomUser",
		Type:        "object",
		Description: "User with custom types",
		Required:    []string{"id", "email"},
		Properties: []generator.Property{
			{
				Name:        "id",
				Type:        generator.PrimitiveType{Name: "string", Format: "uuid"},
				Description: "UUID identifier",
				Required:    true,
			},
			{
				Name:        "email",
				Type:        generator.PrimitiveType{Name: "string", Format: "email"},
				Description: "Email address",
				Required:    true,
			},
		},
	}

	config := generator.Config{
		OutputFolder:   tempDir,
		PackageName:    "custom-types-test",
		TargetLanguage: "typescript",
		ConfigFile:     configPath,
	}

	err := gen.Generate([]generator.DTO{dto}, config)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Check that custom types are used
	userFile := filepath.Join(tempDir, "custom-user.ts")
	content := testutils.ReadFile(t, userFile)

	// Should use custom UUID type
	if !strings.Contains(content, "UUID") {
		t.Errorf("Should use custom UUID type, got content:\n%s", content)
	}

	// Should use custom Email type
	if !strings.Contains(content, "EmailString") {
		t.Errorf("Should use custom EmailString type, got content:\n%s", content)
	}

	// Should have custom imports
	if !strings.Contains(content, "import { UUID } from './custom-types';") {
		t.Errorf("Should have UUID import, got content:\n%s", content)
	}
	if !strings.Contains(content, "import { EmailString } from './email-utils';") {
		t.Errorf("Should have EmailString import, got content:\n%s", content)
	}
}
