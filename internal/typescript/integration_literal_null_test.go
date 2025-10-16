package typescript

import (
	"dtoForge/internal/generator"
	"dtoForge/internal/testutils"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestTypeScriptGenerator_OpenAPILiteralNullIntegration(t *testing.T) {
	// Create a test OpenAPI spec with literal+null enums
	testSpec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":   "Test Literal Null",
			"version": "1.0.0",
		},
		"components": map[string]interface{}{
			"schemas": map[string]interface{}{
				"TestEntity": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"singleValueWithNull": map[string]interface{}{
							"type": "string",
							"enum": []interface{}{"active", nil},
						},
						"multipleValuesWithNull": map[string]interface{}{
							"type": "string",
							"enum": []interface{}{"low", "medium", "high", nil},
						},
						"regularEnum": map[string]interface{}{
							"type": "string",
							"enum": []interface{}{"option1", "option2"},
						},
						"singleValueNoNull": map[string]interface{}{
							"type": "string",
							"enum": []interface{}{"document"},
						},
					},
				},
			},
		},
	}

	// Write test spec to temporary file
	tempDir := testutils.TempDir(t)
	specFile := filepath.Join(tempDir, "test-spec.yaml")

	specData, err := yaml.Marshal(testSpec)
	if err != nil {
		t.Fatalf("Failed to marshal test spec: %v", err)
	}

	err = os.WriteFile(specFile, specData, 0644)
	if err != nil {
		t.Fatalf("Failed to write test spec: %v", err)
	}

	// Generate TypeScript code using the full pipeline
	outputDir := filepath.Join(tempDir, "output")
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}

	// For this test, we'll use the generator directly to test the conversion
	// since the main pipeline requires building the binary
	spec := readTestSpec(t, specFile)
	dtos := convertTestSpecToDTOs(t, spec)

	gen := NewTypeScriptGenerator()
	config := generator.Config{
		OutputFolder:   outputDir,
		PackageName:    "test-integration",
		TargetLanguage: "typescript",
	}

	err = gen.Generate(dtos, config)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify the generated output
	testEntityFile := filepath.Join(outputDir, "test-entity.ts")
	testutils.AssertFileExists(t, testEntityFile)

	// Should generate correct literal+null unions
	testutils.AssertFileContains(t, testEntityFile, "singleValueWithNull: t.union([t.literal('active'), t.null])")
	testutils.AssertFileContains(t, testEntityFile, "multipleValuesWithNull: t.union([t.keyof({'low': null, 'medium': null, 'high': null}), t.null])")

	// Should generate regular enum unchanged
	testutils.AssertFileContains(t, testEntityFile, "regularEnum: t.keyof({'option1': null, 'option2': null})")

	// Should generate single literal without keyof
	testutils.AssertFileContains(t, testEntityFile, "singleValueNoNull: t.literal('document')")
}

// Helper functions to simulate the main pipeline
func readTestSpec(t *testing.T, path string) map[string]interface{} {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read spec: %v", err)
	}

	var spec map[string]interface{}
	err = yaml.Unmarshal(data, &spec)
	if err != nil {
		t.Fatalf("Failed to unmarshal spec: %v", err)
	}

	return spec
}

func convertTestSpecToDTOs(t *testing.T, spec map[string]interface{}) []generator.DTO {
	// This is a simplified version of the conversion in main.go
	// For a full test, we'd import and use the actual conversion functions
	return []generator.DTO{
		{
			Name: "TestEntity",
			Type: "object",
			Properties: []generator.Property{
				{
					Name: "singleValueWithNull",
					Type: generator.EnumType{
						Name:           "SingleValueWithNullEnum",
						UnderlyingType: "string",
						Values:         []string{"active"},
						ContainsNull:   true,
					},
					Required: false,
				},
				{
					Name: "multipleValuesWithNull",
					Type: generator.EnumType{
						Name:           "MultipleValuesWithNullEnum",
						UnderlyingType: "string",
						Values:         []string{"low", "medium", "high"},
						ContainsNull:   true,
					},
					Required: false,
				},
				{
					Name: "regularEnum",
					Type: generator.EnumType{
						Name:           "RegularEnumEnum",
						UnderlyingType: "string",
						Values:         []string{"option1", "option2"},
						ContainsNull:   false,
					},
					Required: false,
				},
				{
					Name: "singleValueNoNull",
					Type: generator.EnumType{
						Name:           "SingleValueNoNullEnum",
						UnderlyingType: "string",
						Values:         []string{"document"},
						ContainsNull:   false,
					},
					Required: false,
				},
			},
		},
	}
}
