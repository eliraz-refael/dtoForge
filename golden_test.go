package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dtoForge/internal/generator"
	"dtoForge/internal/testutils"
	"dtoForge/internal/typescript"
)

func TestGoldenFiles(t *testing.T) {
	testCases := []struct {
		name        string
		openAPIFile string
		configFile  string
		goldenDir   string
	}{
		{
			name:        "basic-schemas",
			openAPIFile: "testdata/basic-api.yaml",
			goldenDir:   "testdata/golden/basic-schemas",
		},
		{
			name:        "custom-formats",
			openAPIFile: "testdata/formats-api.yaml",
			configFile:  "testdata/custom-formats.config.yaml",
			goldenDir:   "testdata/golden/custom-formats",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := testutils.TempDir(t)
			outputDir := filepath.Join(tempDir, "output")

			// Create output directory
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				t.Fatalf("Failed to create output directory: %v", err)
			}

			// Run the same generation logic as main()
			if err := runDtoForgeGeneration(tc.openAPIFile, outputDir, tc.configFile); err != nil {
				t.Fatalf("Generation failed: %v", err)
			}

			// Compare with golden files
			compareWithGolden(t, outputDir, tc.goldenDir)
		})
	}
}

// runDtoForgeGeneration performs the same logic as main() but in a testable way
func runDtoForgeGeneration(openAPIFile, outputDir, configFile string) error {
	// Read and parse OpenAPI spec
	spec, err := readOpenAPISpec(openAPIFile)
	if err != nil {
		return err
	}

	// Convert to generator DTOs
	dtos, err := convertToGeneratorDTOs(spec)
	if err != nil {
		return err
	}

	if len(dtos) == 0 {
		return nil // No schemas to generate
	}

	// Create TypeScript generator
	tsGen := typescript.NewTypeScriptGenerator()

	// Generate code
	genConfig := generator.Config{
		OutputFolder:   outputDir,
		PackageName:    "generated-schemas",
		TargetLanguage: "typescript",
		ConfigFile:     configFile,
	}

	return tsGen.Generate(dtos, genConfig)
}

func diffLinesSimple(golden, output string, filename string) string {
	goldenLines := strings.Split(golden, "\n")
	outputLines := strings.Split(output, "\n")

	var diff strings.Builder
	diff.WriteString(fmt.Sprintf("=== DIFF for %s ===\n", filename))

	maxLines := len(goldenLines)
	if len(outputLines) > maxLines {
		maxLines = len(outputLines)
	}

	for i := 0; i < maxLines; i++ {
		var goldenLine, outputLine string

		if i < len(goldenLines) {
			goldenLine = goldenLines[i]
		}
		if i < len(outputLines) {
			outputLine = outputLines[i]
		}

		if goldenLine != outputLine {
			diff.WriteString(fmt.Sprintf("Line %d:\n", i+1))
			diff.WriteString(fmt.Sprintf("  Expected: %s\n", goldenLine))
			diff.WriteString(fmt.Sprintf("  Actual:   %s\n", outputLine))
			diff.WriteString("\n")
		}
	}

	return diff.String()
}

// compareWithGolden compares output with golden files and shows diffs
func compareWithGolden(t *testing.T, outputDir, goldenDir string) {
	// Walk through all files in the golden directory
	err := filepath.Walk(goldenDir, func(goldenPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path from golden directory
		relPath, err := filepath.Rel(goldenDir, goldenPath)
		if err != nil {
			t.Errorf("Failed to get relative path: %v", err)
			return nil
		}

		// Corresponding output file
		outputPath := filepath.Join(outputDir, relPath)

		// Check if output file exists
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Errorf("Expected output file %s does not exist", relPath)
			return nil
		}

		// Read both files
		goldenContent := testutils.ReadFile(t, goldenPath)
		outputContent := testutils.ReadFile(t, outputPath)

		// Compare content
		if goldenContent != outputContent {
			diff := diffLinesSimple(goldenContent, outputContent, relPath)
			t.Errorf("File %s differs from golden file:\n%s", relPath, diff)
		}

		return nil
	})

	if err != nil {
		t.Errorf("Error walking golden directory: %v", err)
	}

	// Also check if there are any extra files in output that shouldn't be there
	err = filepath.Walk(outputDir, func(outputPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path from output directory
		relPath, err := filepath.Rel(outputDir, outputPath)
		if err != nil {
			t.Errorf("Failed to get relative path: %v", err)
			return nil
		}

		// Corresponding golden file
		goldenPath := filepath.Join(goldenDir, relPath)

		// Check if golden file exists
		if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
			t.Errorf("Unexpected output file %s (no corresponding golden file)", relPath)
		}

		return nil
	})

	if err != nil {
		t.Errorf("Error walking output directory: %v", err)
	}
}

// updateGoldenFiles is a helper function to update golden files when the expected output changes
// Run with: UPDATE_GOLDEN=true go test -run TestGoldenFilesWithUpdate
func updateGoldenFiles(t *testing.T, outputDir, goldenDir string) {
	// Remove existing golden directory
	os.RemoveAll(goldenDir)

	// Create golden directory
	if err := os.MkdirAll(goldenDir, 0755); err != nil {
		t.Fatalf("Failed to create golden directory: %v", err)
	}

	// Copy all files from output to golden
	err := filepath.Walk(outputDir, func(outputPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(outputDir, outputPath)
		if err != nil {
			return err
		}

		// Target golden file path
		goldenPath := filepath.Join(goldenDir, relPath)

		// Create directory if needed
		goldenFileDir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(goldenFileDir, 0755); err != nil {
			return err
		}

		// Copy file content using ioutil.WriteFile directly
		content := testutils.ReadFile(t, outputPath)
		if err := os.WriteFile(goldenPath, []byte(content), 0644); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to update golden files: %v", err)
	}
}

func TestGoldenFilesWithUpdate(t *testing.T) {
	// Check if update flag is set
	updateGolden := os.Getenv("UPDATE_GOLDEN") == "true"

	testCases := []struct {
		name        string
		openAPIFile string
		configFile  string
		goldenDir   string
	}{
		{
			name:        "basic-schemas",
			openAPIFile: "testdata/basic-api.yaml",
			goldenDir:   "testdata/golden/basic-schemas",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := testutils.TempDir(t)
			outputDir := filepath.Join(tempDir, "output")

			if err := os.MkdirAll(outputDir, 0755); err != nil {
				t.Fatalf("Failed to create output directory: %v", err)
			}

			if err := runDtoForgeGeneration(tc.openAPIFile, outputDir, tc.configFile); err != nil {
				t.Fatalf("Generation failed: %v", err)
			}

			if updateGolden {
				updateGoldenFiles(t, outputDir, tc.goldenDir)
				t.Log("Updated golden files for", tc.name)
				return
			} else {
				compareWithGolden(t, outputDir, tc.goldenDir)
			}
		})
	}
}
