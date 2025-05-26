package testutils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dtoForge/internal/generator"
)

// TestCase represents a single test scenario
type TestCase struct {
	Name        string
	OpenAPISpec string
	Config      string // YAML config content (optional)
	Expected    map[string]string // filename -> expected content
}

// TempDir creates a temporary directory for testing
func TempDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "dtoforge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	return dir
}

// WriteFile writes content to a file in the test directory
func WriteFile(t *testing.T, dir, filename, content string) string {
	path := filepath.Join(dir, filename)
	if err := ioutil.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file %s: %v", filename, err)
	}
	return path
}

// ReadFile reads content from a generated file
func ReadFile(t *testing.T, path string) string {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	return string(content)
}

// AssertFileExists checks if a file exists
func AssertFileExists(t *testing.T, path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist", path)
	}
}

// AssertFileContains checks if a file contains expected content
func AssertFileContains(t *testing.T, path, expected string) {
	content := ReadFile(t, path)
	if !strings.Contains(content, expected) {
		t.Errorf("File %s does not contain expected content:\nExpected: %s\nActual: %s",
			path, expected, content)
	}
}

// AssertFileNotContains checks if a file does NOT contain content
func AssertFileNotContains(t *testing.T, path, unexpected string) {
	content := ReadFile(t, path)
	if strings.Contains(content, unexpected) {
		t.Errorf("File %s contains unexpected content: %s", path, unexpected)
	}
}

// NormalizeWhitespace removes extra whitespace for easier comparison
func NormalizeWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	var normalized []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return strings.Join(normalized, "\n")
}

// CreateTestDTO creates a simple test DTO for unit tests
func CreateTestDTO(name string) generator.DTO {
	return generator.DTO{
		Name:        name,
		Description: "Test DTO",
		Type:        "object",
		Required:    []string{"id"},
		Properties: []generator.Property{
			{
				Name:        "id",
				Type:        generator.PrimitiveType{Name: "string"},
				Description: "Identifier",
				Required:    true,
			},
			{
				Name:        "name",
				Type:        generator.PrimitiveType{Name: "string"},
				Description: "Name",
				Required:    false,
			},
		},
		Metadata: make(map[string]string),
	}
}
