package typescript

import (
	"dtoForge/internal/generator"
	"dtoForge/internal/testutils"
	"path/filepath"
	"testing"
)

func TestTypeScriptGenerator_OneOfSupport(t *testing.T) {
	gen := NewTypeScriptGenerator()
	tempDir := testutils.TempDir(t)

	// Create DTOs with oneOf (union type)
	dtos := []generator.DTO{
		{
			Name:        "ActivityTypeAssets",
			Type:        "object",
			Description: "Activity with asset IDs",
			Required:    []string{"assetIds"},
			Properties: []generator.Property{
				{
					Name:     "assetIds",
					Type:     generator.ArrayType{ElementType: generator.PrimitiveType{Name: "string"}},
					Required: true,
				},
			},
		},
		{
			Name:        "ActivityTypeEntities",
			Type:        "object",
			Description: "Activity with entity IDs",
			Required:    []string{"entityIds"},
			Properties: []generator.Property{
				{
					Name:     "entityIds",
					Type:     generator.ArrayType{ElementType: generator.PrimitiveType{Name: "string"}},
					Required: true,
				},
			},
		},
		{
			Name:        "ActivityTypeNone",
			Type:        "object",
			Description: "Empty activity type",
			Properties: []generator.Property{
				{
					Name:     "placeholder",
					Type:     generator.PrimitiveType{Name: "string"},
					Required: false,
				},
			},
		},
		{
			Name:        "Activity",
			Type:        "object",
			Description: "Activity with oneOf field",
			Required:    []string{"index"},
			Properties: []generator.Property{
				{
					Name:        "index",
					Type:        generator.PrimitiveType{Name: "integer"},
					Description: "Activity index",
					Required:    true,
				},
				{
					Name: "methodParams",
					Type: generator.UnionType{
						Types: []generator.IRType{
							generator.ReferenceType{RefName: "ActivityTypeAssets"},
							generator.ReferenceType{RefName: "ActivityTypeEntities"},
							generator.ReferenceType{RefName: "ActivityTypeNone"},
						},
					},
					Description: "OneOf multiple activity types",
					Required:    false,
				},
			},
		},
	}

	config := generator.Config{
		OutputFolder:   tempDir,
		PackageName:    "test-oneof",
		TargetLanguage: "typescript",
	}

	err := gen.Generate(dtos, config)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Check generated file
	activityFile := filepath.Join(tempDir, "activity.ts")
	testutils.AssertFileExists(t, activityFile)

	// Check that oneOf generates t.union
	testutils.AssertFileContains(t, activityFile, "methodParams: t.union([")
	testutils.AssertFileContains(t, activityFile, "ActivityTypeAssets")
	testutils.AssertFileContains(t, activityFile, "ActivityTypeEntities")
	testutils.AssertFileContains(t, activityFile, "ActivityTypeNone")
}

func TestTypeScriptGenerator_OneOfWithPrimitives(t *testing.T) {
	gen := NewTypeScriptGenerator()
	tempDir := testutils.TempDir(t)

	// Test oneOf with primitive types (string | number | boolean)
	dto := generator.DTO{
		Name:        "FlexibleValue",
		Type:        "object",
		Description: "Object with flexible value type",
		Properties: []generator.Property{
			{
				Name: "value",
				Type: generator.UnionType{
					Types: []generator.IRType{
						generator.PrimitiveType{Name: "string"},
						generator.PrimitiveType{Name: "number"},
						generator.PrimitiveType{Name: "boolean"},
					},
				},
				Description: "Can be string, number, or boolean",
				Required:    false,
			},
		},
	}

	config := generator.Config{
		OutputFolder:   tempDir,
		PackageName:    "test-oneof-primitives",
		TargetLanguage: "typescript",
	}

	err := gen.Generate([]generator.DTO{dto}, config)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Check generated file
	flexibleFile := filepath.Join(tempDir, "flexible-value.ts")
	testutils.AssertFileExists(t, flexibleFile)

	// Check that oneOf with primitives generates t.union
	testutils.AssertFileContains(t, flexibleFile, "value: t.union([")
	testutils.AssertFileContains(t, flexibleFile, "t.string")
	testutils.AssertFileContains(t, flexibleFile, "t.number")
	testutils.AssertFileContains(t, flexibleFile, "t.boolean")
}

func TestTypeScriptGenerator_OneOfNullable(t *testing.T) {
	gen := NewTypeScriptGenerator()
	tempDir := testutils.TempDir(t)

	// Test oneOf with x-nullable
	dto := generator.DTO{
		Name:     "NullableUnion",
		Type:     "object",
		Required: []string{"requiredUnion"},
		Properties: []generator.Property{
			{
				Name: "optionalUnion",
				Type: generator.UnionType{
					Types: []generator.IRType{
						generator.PrimitiveType{Name: "string"},
						generator.PrimitiveType{Name: "number"},
					},
				},
				Extensions: map[string]interface{}{
					"x-nullable": true,
				},
				Required: false,
			},
			{
				Name: "requiredUnion",
				Type: generator.UnionType{
					Types: []generator.IRType{
						generator.ReferenceType{RefName: "TypeA"},
						generator.ReferenceType{RefName: "TypeB"},
					},
				},
				Extensions: map[string]interface{}{
					"x-nullable": true,
				},
				Required: true,
			},
		},
	}

	config := generator.Config{
		OutputFolder:   tempDir,
		PackageName:    "test-oneof-nullable",
		TargetLanguage: "typescript",
	}

	err := gen.Generate([]generator.DTO{dto}, config)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	nullableFile := filepath.Join(tempDir, "nullable-union.ts")

	// oneOf with x-nullable should wrap the entire union in another union with null
	testutils.AssertFileContains(t, nullableFile, "optionalUnion: t.union([t.union([")
	testutils.AssertFileContains(t, nullableFile, "t.null])")
	testutils.AssertFileContains(t, nullableFile, "requiredUnion: t.union([t.union([")
}
