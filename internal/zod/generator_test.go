package zod

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dtoForge/internal/generator"
	"dtoForge/internal/testutils"
)

func TestZodGenerator_Language(t *testing.T) {
	gen := NewZodGenerator()
	if got := gen.Language(); got != "typescript-zod" {
		t.Errorf("Language() = %v, want %v", got, "typescript-zod")
	}
}

func TestZodGenerator_FileExtension(t *testing.T) {
	gen := NewZodGenerator()
	if got := gen.FileExtension(); got != ".ts" {
		t.Errorf("FileExtension() = %v, want %v", got, ".ts")
	}
}

func TestZodGenerator_ToZodType(t *testing.T) {
	gen := NewZodGenerator()
	gen.customTypes = NewCustomTypeRegistry()

	tests := []struct {
		name     string
		irType   generator.IRType
		nullable bool
		optional bool
		expected string
	}{
		{
			name:     "Basic string",
			irType:   generator.PrimitiveType{Name: "string"},
			nullable: false,
			optional: false,
			expected: "z.string()",
		},
		{
			name:     "String with email format",
			irType:   generator.PrimitiveType{Name: "string", Format: "email"},
			nullable: false,
			optional: false,
			expected: "z.string().email()",
		},
		{
			name:     "UUID with format",
			irType:   generator.PrimitiveType{Name: "string", Format: "uuid"},
			nullable: false,
			optional: false,
			expected: "z.string().uuid()",
		},
		{
			name:     "Optional string",
			irType:   generator.PrimitiveType{Name: "string"},
			nullable: false,
			optional: true,
			expected: "z.string().optional()",
		},
		{
			name:     "Nullable string",
			irType:   generator.PrimitiveType{Name: "string"},
			nullable: true,
			optional: false,
			expected: "z.string().nullable()",
		},
		{
			name:     "Nullable and optional string",
			irType:   generator.PrimitiveType{Name: "string"},
			nullable: true,
			optional: true,
			expected: "z.string().nullable().optional()",
		},
		{
			name:     "Number type",
			irType:   generator.PrimitiveType{Name: "number"},
			nullable: false,
			optional: false,
			expected: "z.number()",
		},
		{
			name:     "Boolean type",
			irType:   generator.PrimitiveType{Name: "boolean"},
			nullable: false,
			optional: false,
			expected: "z.boolean()",
		},
		{
			name:     "Array of strings",
			irType:   generator.ArrayType{ElementType: generator.PrimitiveType{Name: "string"}},
			nullable: false,
			optional: false,
			expected: "z.array(z.string())",
		},
		{
			name:     "Reference type",
			irType:   generator.ReferenceType{RefName: "User"},
			nullable: false,
			optional: false,
			expected: "UserSchema",
		},
		{
			name:     "Enum type",
			irType:   generator.EnumType{Values: []string{"active", "inactive"}},
			nullable: false,
			optional: false,
			expected: "z.enum(['active', 'inactive'])",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gen.toZodType(tt.irType, tt.nullable, tt.optional)
			if got != tt.expected {
				t.Errorf("toZodType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestZodGenerator_PrimitiveToZod(t *testing.T) {
	gen := NewZodGenerator()
	gen.customTypes = NewCustomTypeRegistry()

	tests := []struct {
		name     string
		primType generator.PrimitiveType
		expected string
	}{
		{"String", generator.PrimitiveType{Name: "string"}, "z.string()"},
		{"String with email", generator.PrimitiveType{Name: "string", Format: "email"}, "z.string().email()"},
		{"String with uuid", generator.PrimitiveType{Name: "string", Format: "uuid"}, "z.string().uuid()"},
		{"String with date-time", generator.PrimitiveType{Name: "string", Format: "date-time"}, "z.string().datetime()"},
		{"Number", generator.PrimitiveType{Name: "number"}, "z.number()"},
		{"Integer", generator.PrimitiveType{Name: "integer"}, "z.number()"},
		{"Boolean", generator.PrimitiveType{Name: "boolean"}, "z.boolean()"},
		{"Unknown", generator.PrimitiveType{Name: "unknown"}, "z.unknown()"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gen.primitiveToZod(tt.primType)
			if got != tt.expected {
				t.Errorf("primitiveToZod() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestZodGenerator_StringWithFormat(t *testing.T) {
	gen := NewZodGenerator()
	gen.customTypes = NewCustomTypeRegistry()

	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{"Email format", "email", "z.string().email()"},
		{"UUID format", "uuid", "z.string().uuid()"},
		{"URI format", "uri", "z.string().url()"},
		{"URL format", "url", "z.string().url()"},
		{"Date-time format", "date-time", "z.string().datetime()"},
		{"Date format", "date", "z.string().date()"},
		{"No format", "", "z.string()"},
		{"Unknown format", "custom-format", "z.string() /* format: custom-format */"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gen.stringWithFormat(tt.format)
			if got != tt.expected {
				t.Errorf("stringWithFormat() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestZodGenerator_UtilityFunctions(t *testing.T) {
	gen := NewZodGenerator()

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

func TestZodGenerator_HasDescription(t *testing.T) {
	gen := NewZodGenerator()

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

func TestZodGenerator_Generate_MultipleFiles(t *testing.T) {
	gen := NewZodGenerator()
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
		PackageName:    "test-zod",
		TargetLanguage: "typescript-zod",
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
	testutils.AssertFileContains(t, userFile, "export const UserSchema = z.object({")
	testutils.AssertFileContains(t, userFile, "export type User = z.infer<typeof UserSchema>;")
	testutils.AssertFileContains(t, userFile, "import { z } from 'zod';")

	// Check content of status.ts (enum)
	statusFile := filepath.Join(tempDir, "status.ts")
	testutils.AssertFileContains(t, statusFile, "export const StatusSchema = z.enum([")
	testutils.AssertFileContains(t, statusFile, "'active',")
	testutils.AssertFileContains(t, statusFile, "'inactive',")
	testutils.AssertFileContains(t, statusFile, "'pending'")

	// Check index.ts
	indexFile := filepath.Join(tempDir, "index.ts")
	testutils.AssertFileContains(t, indexFile, "export * from './user';")
	testutils.AssertFileContains(t, indexFile, "export * from './status';")
	testutils.AssertFileContains(t, indexFile, "export { z } from 'zod';")

	// Check package.json
	packageFile := filepath.Join(tempDir, "package.json")
	testutils.AssertFileContains(t, packageFile, `"zod": "^3.22.4"`)
	testutils.AssertFileContains(t, packageFile, `"name": "test-zod"`)
}

func TestZodGenerator_Generate_SingleFile(t *testing.T) {
	gen := NewZodGenerator()
	tempDir := testutils.TempDir(t)

	// Create a config file for single file mode
	configContent := `typescript-zod:
  output:
    mode: single
    singleFileName: schemas.ts
  generation:
    generatePackageJson: false
    generateHelpers: true`

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
		TargetLanguage: "typescript-zod",
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
	if !strings.Contains(content, "export const UserSchema") {
		t.Error("Single file should contain UserSchema")
	}
	if !strings.Contains(content, "export const StatusSchema") {
		t.Error("Single file should contain StatusSchema")
	}

	// Should contain helper functions
	if !strings.Contains(content, "export const validateData") {
		t.Error("Single file should contain validateData helper")
	}
}

func TestZodGenerator_CustomTypes(t *testing.T) {
	gen := NewZodGenerator()
	tempDir := testutils.TempDir(t)

	// Create config with custom types
	configContent := `typescript-zod:
  customTypes:
    uuid:
      zodType: "z.string().uuid().brand('UUID')"
      typeScriptType: "UUID"
      import: "import { UUID } from './custom-types';"
    email:
      zodType: "EmailSchema"
      typeScriptType: "Email"
      import: "import { EmailSchema } from './email-utils';"`

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
		TargetLanguage: "typescript-zod",
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
	if !strings.Contains(content, "z.string().uuid().brand('UUID')") {
		t.Errorf("Should use custom UUID type, got content:\n%s", content)
	}

	// Should use custom Email type
	if !strings.Contains(content, "EmailSchema") {
		t.Errorf("Should use custom EmailSchema, got content:\n%s", content)
	}

	// Should have custom imports
	if !strings.Contains(content, "import { UUID } from './custom-types';") {
		t.Errorf("Should have UUID import, got content:\n%s", content)
	}
	if !strings.Contains(content, "import { EmailSchema } from './email-utils';") {
		t.Errorf("Should have EmailSchema import, got content:\n%s", content)
	}
}
