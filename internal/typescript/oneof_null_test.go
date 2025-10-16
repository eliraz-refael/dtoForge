package typescript

import (
	"dtoForge/internal/generator"
	"dtoForge/internal/testutils"
	"path/filepath"
	"testing"
)

func TestTypeScriptGenerator_OneOfWithNull(t *testing.T) {
	gen := NewTypeScriptGenerator()
	tempDir := testutils.TempDir(t)

	// Test various oneOf + null scenarios
	dtos := []generator.DTO{
		{
			Name:        "User",
			Type:        "object",
			Description: "User reference type",
			Properties: []generator.Property{
				{
					Name:     "id",
					Type:     generator.PrimitiveType{Name: "string"},
					Required: true,
				},
				{
					Name:     "name",
					Type:     generator.PrimitiveType{Name: "string"},
					Required: true,
				},
			},
		},
		{
			Name:        "Admin",
			Type:        "object",
			Description: "Admin reference type",
			Properties: []generator.Property{
				{
					Name:     "id",
					Type:     generator.PrimitiveType{Name: "string"},
					Required: true,
				},
				{
					Name:     "permissions",
					Type:     generator.ArrayType{ElementType: generator.PrimitiveType{Name: "string"}},
					Required: true,
				},
			},
		},
		{
			Name:        "TestEntity",
			Type:        "object",
			Description: "Test oneOf with null scenarios",
			Properties: []generator.Property{
				{
					Name: "nullableUser",
					Type: generator.UnionType{
						Types: []generator.IRType{
							generator.ReferenceType{RefName: "User"},
							generator.NullType{},
						},
					},
					Description: "OneOf with reference and null",
					Required:    false,
				},
				{
					Name: "nullableUserOrAdmin",
					Type: generator.UnionType{
						Types: []generator.IRType{
							generator.ReferenceType{RefName: "User"},
							generator.ReferenceType{RefName: "Admin"},
							generator.NullType{},
						},
					},
					Description: "OneOf with multiple references and null",
					Required:    false,
				},
				{
					Name: "nullableString",
					Type: generator.UnionType{
						Types: []generator.IRType{
							generator.PrimitiveType{Name: "string"},
							generator.NullType{},
						},
					},
					Description: "OneOf with primitive and null",
					Required:    false,
				},
				{
					Name: "regularUnion",
					Type: generator.UnionType{
						Types: []generator.IRType{
							generator.PrimitiveType{Name: "string"},
							generator.PrimitiveType{Name: "number"},
						},
					},
					Description: "Regular union without null",
					Required:    false,
				},
			},
		},
	}

	config := generator.Config{
		OutputFolder:   tempDir,
		PackageName:    "test-oneof-null",
		TargetLanguage: "typescript",
	}

	err := gen.Generate(dtos, config)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Check generated file
	testEntityFile := filepath.Join(tempDir, "test-entity.ts")
	testutils.AssertFileExists(t, testEntityFile)

	// Test oneOf with reference + null generates proper union
	testutils.AssertFileContains(t, testEntityFile, "nullableUser: t.union([User, t.null])")

	// Test oneOf with multiple references + null
	testutils.AssertFileContains(t, testEntityFile, "nullableUserOrAdmin: t.union([User, Admin, t.null])")

	// Test oneOf with primitive + null
	testutils.AssertFileContains(t, testEntityFile, "nullableString: t.union([t.string, t.null])")

	// Test regular union without null (no t.null)
	testutils.AssertFileContains(t, testEntityFile, "regularUnion: t.union([t.string, t.number])")
	testutils.AssertFileNotContains(t, testEntityFile, "regularUnion: t.union([t.string, t.number, t.null])")
}

func TestTypeScriptGenerator_NullTypeStandalone(t *testing.T) {
	gen := NewTypeScriptGenerator()
	tempDir := testutils.TempDir(t)

	// Test standalone null type handling
	dto := generator.DTO{
		Name:        "NullTest",
		Type:        "object",
		Description: "Test standalone null type",
		Properties: []generator.Property{
			{
				Name:        "nullOnly",
				Type:        generator.NullType{},
				Description: "Only null type",
				Required:    false,
			},
		},
	}

	config := generator.Config{
		OutputFolder:   tempDir,
		PackageName:    "test-null-standalone",
		TargetLanguage: "typescript",
	}

	err := gen.Generate([]generator.DTO{dto}, config)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Check that standalone null type generates t.null
	nullTestFile := filepath.Join(tempDir, "null-test.ts")
	testutils.AssertFileExists(t, nullTestFile)
	testutils.AssertFileContains(t, nullTestFile, "nullOnly: t.null")
}
