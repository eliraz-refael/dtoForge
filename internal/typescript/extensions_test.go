package typescript

import (
	"dtoForge/internal/generator"
	"dtoForge/internal/testutils"
	"path/filepath"
	"testing"
)

func TestTypeScriptGenerator_XNullableExtension(t *testing.T) {
	gen := NewTypeScriptGenerator()
	tempDir := testutils.TempDir(t)

	// Create a DTO with x-nullable extension
	dto := generator.DTO{
		Name:        "User",
		Type:        "object",
		Description: "User with nullable fields",
		Required:    []string{"id"},
		Properties: []generator.Property{
			{
				Name:        "id",
				Type:        generator.PrimitiveType{Name: "string"},
				Description: "User ID",
				Required:    true,
			},
			{
				Name:        "email",
				Type:        generator.PrimitiveType{Name: "string", Format: "email"},
				Description: "User email (nullable via x-nullable)",
				Required:    false,
				Extensions: map[string]interface{}{
					"x-nullable": true,
				},
			},
			{
				Name:        "phone",
				Type:        generator.PrimitiveType{Name: "string"},
				Description: "Phone number (not nullable)",
				Required:    false,
				Extensions: map[string]interface{}{
					"x-nullable": false,
				},
			},
			{
				Name:        "address",
				Type:        generator.PrimitiveType{Name: "string"},
				Description: "Address (no x-nullable)",
				Required:    false,
			},
		},
	}

	config := generator.Config{
		OutputFolder:   tempDir,
		PackageName:    "test-extensions",
		TargetLanguage: "typescript",
	}

	err := gen.Generate([]generator.DTO{dto}, config)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Check generated file
	userFile := filepath.Join(tempDir, "user.ts")
	testutils.AssertFileExists(t, userFile)

	// Check that email is nullable (x-nullable: true)
	testutils.AssertFileContains(t, userFile, "email: t.union([t.string, t.null])")

	// Check that phone is not nullable (x-nullable: false)
	testutils.AssertFileContains(t, userFile, "phone: t.string")

	// Check that address is not nullable (no x-nullable)
	testutils.AssertFileContains(t, userFile, "address: t.string")
}

func TestTypeScriptGenerator_XNullableWithRequired(t *testing.T) {
	gen := NewTypeScriptGenerator()
	tempDir := testutils.TempDir(t)

	// Test x-nullable on required fields
	dto := generator.DTO{
		Name:        "Product",
		Type:        "object",
		Description: "Product with nullable required field",
		Required:    []string{"name", "description"},
		Properties: []generator.Property{
			{
				Name:        "name",
				Type:        generator.PrimitiveType{Name: "string"},
				Description: "Product name (required but nullable)",
				Required:    true,
				Extensions: map[string]interface{}{
					"x-nullable": true,
				},
			},
			{
				Name:        "description",
				Type:        generator.PrimitiveType{Name: "string"},
				Description: "Product description (required, not nullable)",
				Required:    true,
			},
		},
	}

	config := generator.Config{
		OutputFolder:   tempDir,
		PackageName:    "test-nullable-required",
		TargetLanguage: "typescript",
	}

	err := gen.Generate([]generator.DTO{dto}, config)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	productFile := filepath.Join(tempDir, "product.ts")

	// Required + x-nullable should be: field: t.union([t.string, t.null])
	testutils.AssertFileContains(t, productFile, "name: t.union([t.string, t.null])")

	// Required without x-nullable should be: field: t.string
	testutils.AssertFileContains(t, productFile, "description: t.string")
}

func TestTypeScriptGenerator_XNullableConfig(t *testing.T) {
	gen := NewTypeScriptGenerator()
	tempDir := testutils.TempDir(t)

	// Create config that disables x-nullable handling
	configContent := `extensions:
  x-nullable:
    enabled: false`

	configPath := testutils.WriteFile(t, tempDir, "config.yaml", configContent)

	dto := generator.DTO{
		Name:     "Settings",
		Type:     "object",
		Required: []string{},
		Properties: []generator.Property{
			{
				Name:     "theme",
				Type:     generator.PrimitiveType{Name: "string"},
				Required: false,
				Extensions: map[string]interface{}{
					"x-nullable": true, // Should be ignored when disabled
				},
			},
		},
	}

	config := generator.Config{
		OutputFolder:   tempDir,
		PackageName:    "test-disabled-nullable",
		TargetLanguage: "typescript",
		ConfigFile:     configPath,
	}

	err := gen.Generate([]generator.DTO{dto}, config)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	settingsFile := filepath.Join(tempDir, "settings.ts")

	// With x-nullable disabled, should generate normal optional
	testutils.AssertFileContains(t, settingsFile, "theme: t.string")
	testutils.AssertFileNotContains(t, settingsFile, "t.union([t.string, t.null])")
}

func TestTypeScriptGenerator_XNullableWithArrays(t *testing.T) {
	gen := NewTypeScriptGenerator()
	tempDir := testutils.TempDir(t)

	dto := generator.DTO{
		Name:     "Collection",
		Type:     "object",
		Required: []string{},
		Properties: []generator.Property{
			{
				Name: "items",
				Type: generator.ArrayType{
					ElementType: generator.PrimitiveType{Name: "string"},
				},
				Required: false,
				Extensions: map[string]interface{}{
					"x-nullable": true,
				},
			},
			{
				Name: "tags",
				Type: generator.ArrayType{
					ElementType: generator.PrimitiveType{Name: "string"},
				},
				Required: true,
				Extensions: map[string]interface{}{
					"x-nullable": true,
				},
			},
		},
	}

	config := generator.Config{
		OutputFolder:   tempDir,
		PackageName:    "test-nullable-arrays",
		TargetLanguage: "typescript",
	}

	err := gen.Generate([]generator.DTO{dto}, config)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	collectionFile := filepath.Join(tempDir, "collection.ts")

	// Optional array with x-nullable
	testutils.AssertFileContains(t, collectionFile, "items: t.union([t.array(t.string), t.null])")

	// Required array with x-nullable
	testutils.AssertFileContains(t, collectionFile, "tags: t.union([t.array(t.string), t.null])")
}

func TestTypeScriptGenerator_XNullableWithReferences(t *testing.T) {
	gen := NewTypeScriptGenerator()
	tempDir := testutils.TempDir(t)

	dtos := []generator.DTO{
		{
			Name:     "Address",
			Type:     "object",
			Required: []string{"street"},
			Properties: []generator.Property{
				{
					Name:     "street",
					Type:     generator.PrimitiveType{Name: "string"},
					Required: true,
				},
			},
		},
		{
			Name:     "Person",
			Type:     "object",
			Required: []string{},
			Properties: []generator.Property{
				{
					Name:     "homeAddress",
					Type:     generator.ReferenceType{RefName: "Address"},
					Required: false,
					Extensions: map[string]interface{}{
						"x-nullable": true,
					},
				},
				{
					Name:     "workAddress",
					Type:     generator.ReferenceType{RefName: "Address"},
					Required: true,
					Extensions: map[string]interface{}{
						"x-nullable": true,
					},
				},
			},
		},
	}

	config := generator.Config{
		OutputFolder:   tempDir,
		PackageName:    "test-nullable-refs",
		TargetLanguage: "typescript",
	}

	err := gen.Generate(dtos, config)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	personFile := filepath.Join(tempDir, "person.ts")

	// Optional reference with x-nullable
	testutils.AssertFileContains(t, personFile, "homeAddress: t.union([Address, t.null])")

	// Required reference with x-nullable
	testutils.AssertFileContains(t, personFile, "workAddress: t.union([Address, t.null])")
}
