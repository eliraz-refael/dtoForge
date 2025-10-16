package typescript

import (
	"dtoForge/internal/generator"
	"dtoForge/internal/testutils"
	"path/filepath"
	"testing"
)

func TestTypeScriptGenerator_LiteralNullUnions(t *testing.T) {
	gen := NewTypeScriptGenerator()
	tempDir := testutils.TempDir(t)

	// Test various literal+null union scenarios
	dtos := []generator.DTO{
		{
			Name:        "SingleLiteralWithNull",
			Type:        "object",
			Description: "Single literal value with null",
			Properties: []generator.Property{
				{
					Name: "status",
					Type: generator.EnumType{
						Name:           "StatusEnum",
						UnderlyingType: "string",
						Values:         []string{"active"},
						ContainsNull:   true,
					},
					Required: false,
				},
			},
		},
		{
			Name:        "MultipleLiteralsWithNull",
			Type:        "object",
			Description: "Multiple literal values with null",
			Properties: []generator.Property{
				{
					Name: "priority",
					Type: generator.EnumType{
						Name:           "PriorityEnum",
						UnderlyingType: "string",
						Values:         []string{"low", "medium", "high"},
						ContainsNull:   true,
					},
					Required: false,
				},
			},
		},
		{
			Name:        "RegularEnum",
			Type:        "object",
			Description: "Regular enum without null (should be unchanged)",
			Properties: []generator.Property{
				{
					Name: "category",
					Type: generator.EnumType{
						Name:           "CategoryEnum",
						UnderlyingType: "string",
						Values:         []string{"tech", "business", "personal"},
						ContainsNull:   false,
					},
					Required: false,
				},
			},
		},
		{
			Name:        "SingleLiteralNoNull",
			Type:        "object",
			Description: "Single literal without null",
			Properties: []generator.Property{
				{
					Name: "type",
					Type: generator.EnumType{
						Name:           "TypeEnum",
						UnderlyingType: "string",
						Values:         []string{"document"},
						ContainsNull:   false,
					},
					Required: false,
				},
			},
		},
	}

	config := generator.Config{
		OutputFolder:   tempDir,
		PackageName:    "test-literal-null",
		TargetLanguage: "typescript",
	}

	err := gen.Generate(dtos, config)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Test single literal + null generates t.union([t.literal(...), t.null])
	singleFile := filepath.Join(tempDir, "single-literal-with-null.ts")
	testutils.AssertFileExists(t, singleFile)
	testutils.AssertFileContains(t, singleFile, "status: t.union([t.literal('active'), t.null])")

	// Test multiple literals + null generates t.union([t.keyof({...}), t.null])
	multipleFile := filepath.Join(tempDir, "multiple-literals-with-null.ts")
	testutils.AssertFileExists(t, multipleFile)
	testutils.AssertFileContains(t, multipleFile, "priority: t.union([t.keyof({'low': null, 'medium': null, 'high': null}), t.null])")

	// Test regular enum is unchanged (uses t.keyof without union)
	regularFile := filepath.Join(tempDir, "regular-enum.ts")
	testutils.AssertFileExists(t, regularFile)
	testutils.AssertFileContains(t, regularFile, "category: t.keyof({'tech': null, 'business': null, 'personal': null})")
	testutils.AssertFileNotContains(t, regularFile, "t.union")

	// Test single literal without null uses t.literal (not t.keyof)
	singleNoNullFile := filepath.Join(tempDir, "single-literal-no-null.ts")
	testutils.AssertFileExists(t, singleNoNullFile)
	testutils.AssertFileContains(t, singleNoNullFile, "type: t.literal('document')")
	testutils.AssertFileNotContains(t, singleNoNullFile, "t.keyof")
	testutils.AssertFileNotContains(t, singleNoNullFile, "t.union")
}

func TestTypeScriptGenerator_NullHandlingNoDoubleWrap(t *testing.T) {
	gen := NewTypeScriptGenerator()
	tempDir := testutils.TempDir(t)

	// Test that nullable property with enum that contains null doesn't double-wrap
	dto := generator.DTO{
		Name:        "NoDoubleWrap",
		Type:        "object",
		Description: "Test no double wrapping of nullable enum with null",
		Properties: []generator.Property{
			{
				Name: "value",
				Type: generator.EnumType{
					Name:           "ValueEnum",
					UnderlyingType: "string",
					Values:         []string{"option1"},
					ContainsNull:   true,
				},
				Nullable: true, // This should NOT cause double wrapping
				Required: false,
			},
		},
	}

	config := generator.Config{
		OutputFolder:   tempDir,
		PackageName:    "test-no-double-wrap",
		TargetLanguage: "typescript",
	}

	err := gen.Generate([]generator.DTO{dto}, config)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Should generate t.union([t.literal('option1'), t.null]) not t.union([t.union([...]), t.null])
	file := filepath.Join(tempDir, "no-double-wrap.ts")
	testutils.AssertFileExists(t, file)
	testutils.AssertFileContains(t, file, "value: t.union([t.literal('option1'), t.null])")
	testutils.AssertFileNotContains(t, file, "t.union([t.union(")
}
