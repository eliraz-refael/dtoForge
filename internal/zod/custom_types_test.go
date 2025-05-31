package zod

import (
	"testing"

	"dtoForge/internal/testutils"
)

func TestCustomTypeRegistry_DefaultMappings(t *testing.T) {
	registry := NewCustomTypeRegistry()

	tests := []struct {
		format   string
		expected CustomTypeMapping
	}{
		{
			format: "email",
			expected: CustomTypeMapping{
				ZodType:        "z.string().email()",
				TypeScriptType: "string",
				Import:         "",
			},
		},
		{
			format: "uuid",
			expected: CustomTypeMapping{
				ZodType:        "z.string().uuid()",
				TypeScriptType: "string",
				Import:         "",
			},
		},
		{
			format: "date-time",
			expected: CustomTypeMapping{
				ZodType:        "z.string().datetime()",
				TypeScriptType: "string",
				Import:         "",
			},
		},
		{
			format: "url",
			expected: CustomTypeMapping{
				ZodType:        "z.string().url()",
				TypeScriptType: "string",
				Import:         "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			mapping, exists := registry.Get(tt.format)
			if !exists {
				t.Errorf("Expected default mapping for format %s to exist", tt.format)
				return
			}

			if mapping.ZodType != tt.expected.ZodType {
				t.Errorf("ZodType = %v, want %v", mapping.ZodType, tt.expected.ZodType)
			}
			if mapping.TypeScriptType != tt.expected.TypeScriptType {
				t.Errorf("TypeScriptType = %v, want %v", mapping.TypeScriptType, tt.expected.TypeScriptType)
			}
			if mapping.Import != tt.expected.Import {
				t.Errorf("Import = %v, want %v", mapping.Import, tt.expected.Import)
			}
		})
	}
}

func TestCustomTypeRegistry_Register(t *testing.T) {
	registry := NewCustomTypeRegistry()

	customMapping := CustomTypeMapping{
		ZodType:        "z.string().custom().brand('Custom')",
		TypeScriptType: "Custom",
		Import:         "import { Custom } from './custom';",
	}

	registry.Register("custom-format", customMapping)

	mapping, exists := registry.Get("custom-format")
	if !exists {
		t.Error("Expected custom mapping to exist after registration")
		return
	}

	if mapping.ZodType != customMapping.ZodType {
		t.Errorf("ZodType = %v, want %v", mapping.ZodType, customMapping.ZodType)
	}
}

func TestCustomTypeRegistry_GetAllImports(t *testing.T) {
	registry := NewCustomTypeRegistry()

	// Add some custom mappings with imports
	registry.Register("custom1", CustomTypeMapping{
		ZodType: "Custom1",
		Import:  "import { Custom1 } from './custom1';",
	})
	registry.Register("custom2", CustomTypeMapping{
		ZodType: "Custom2",
		Import:  "import { Custom2 } from './custom2';",
	})
	registry.Register("custom3", CustomTypeMapping{
		ZodType: "Custom3",
		Import:  "", // No import
	})

	usedFormats := []string{"custom1", "custom2", "custom3", "email"} // email has no import
	imports := registry.GetAllImports(usedFormats)

	// Should always include Zod import first
	if len(imports) < 1 || imports[0] != "import { z } from 'zod';" {
		t.Error("First import should always be Zod")
	}

	// Should include custom imports (sorted alphabetically)
	expectedImports := []string{
		"import { z } from 'zod';",
		"import { Custom1 } from './custom1';",
		"import { Custom2 } from './custom2';",
	}

	if len(imports) != len(expectedImports) {
		t.Errorf("Expected %d imports, got %d", len(expectedImports), len(imports))
	}

	for i, expected := range expectedImports {
		if i >= len(imports) || imports[i] != expected {
			t.Errorf("Import[%d] = %v, want %v", i, imports[i], expected)
		}
	}
}

func TestCustomTypeRegistry_OutputConfig(t *testing.T) {
	registry := NewCustomTypeRegistry()

	// Test defaults
	config := registry.GetOutputConfig()
	if config.Folder != "./generated" {
		t.Errorf("Default folder = %v, want %v", config.Folder, "./generated")
	}
	if config.Mode != "multiple" {
		t.Errorf("Default mode = %v, want %v", config.Mode, "multiple")
	}
	if config.SingleFileName != "schemas.ts" {
		t.Errorf("Default single file name = %v, want %v", config.SingleFileName, "schemas.ts")
	}

	// Test single file mode
	if registry.IsSingleFileMode() {
		t.Error("Should not be in single file mode by default")
	}

	if registry.GetSingleFileName() != "schemas.ts" {
		t.Errorf("GetSingleFileName() = %v, want %v", registry.GetSingleFileName(), "schemas.ts")
	}
}

func TestCustomTypeRegistry_GenerationConfig(t *testing.T) {
	registry := NewCustomTypeRegistry()

	config := registry.GetGenerationConfig()
	if !config.GeneratePackageJson {
		t.Error("Should generate package.json by default")
	}
	if !config.GenerateHelpers {
		t.Error("Should generate helpers by default")
	}
}

func TestCustomTypeRegistry_LoadFromConfig(t *testing.T) {
	registry := NewCustomTypeRegistry()
	tempDir := testutils.TempDir(t)

	// Test with non-existent config file (should not error)
	err := registry.LoadFromConfig("non-existent.yaml")
	if err != nil {
		t.Errorf("LoadFromConfig with non-existent file should not error: %v", err)
	}

	// Create a test config file
	configContent := `typescript-zod:
  output:
    folder: "./custom-output"
    mode: "single"
    singleFileName: "custom-schemas.ts"
  generation:
    generatePackageJson: false
    generateHelpers: false
  customTypes:
    uuid:
      zodType: "z.string().uuid().brand('UUID')"
      typeScriptType: "UUID"
      import: "import { UUID } from './uuid';"
    custom-date:
      zodType: "DateSchema"
      typeScriptType: "CustomDate"
      import: "import { DateSchema } from './dates';"`

	configPath := testutils.WriteFile(t, tempDir, "test-config.yaml", configContent)

	err = registry.LoadFromConfig(configPath)
	if err != nil {
		t.Fatalf("LoadFromConfig failed: %v", err)
	}

	// Test output config was loaded
	outputConfig := registry.GetOutputConfig()
	if outputConfig.Folder != "./custom-output" {
		t.Errorf("Folder = %v, want %v", outputConfig.Folder, "./custom-output")
	}
	if outputConfig.Mode != "single" {
		t.Errorf("Mode = %v, want %v", outputConfig.Mode, "single")
	}
	if outputConfig.SingleFileName != "custom-schemas.ts" {
		t.Errorf("SingleFileName = %v, want %v", outputConfig.SingleFileName, "custom-schemas.ts")
	}

	// Test generation config was loaded
	genConfig := registry.GetGenerationConfig()
	if genConfig.GeneratePackageJson {
		t.Error("GeneratePackageJson should be false")
	}
	if genConfig.GenerateHelpers {
		t.Error("GenerateHelpers should be false")
	}

	// Test custom types were loaded
	uuidMapping, exists := registry.Get("uuid")
	if !exists {
		t.Error("UUID mapping should exist")
	} else {
		if uuidMapping.ZodType != "z.string().uuid().brand('UUID')" {
			t.Errorf("UUID ZodType = %v, want %v", uuidMapping.ZodType, "z.string().uuid().brand('UUID')")
		}
		if uuidMapping.Import != "import { UUID } from './uuid';" {
			t.Errorf("UUID Import = %v, want %v", uuidMapping.Import, "import { UUID } from './uuid';")
		}
	}

	customDateMapping, exists := registry.Get("custom-date")
	if !exists {
		t.Error("Custom date mapping should exist")
	} else {
		if customDateMapping.ZodType != "DateSchema" {
			t.Errorf("CustomDate ZodType = %v, want %v", customDateMapping.ZodType, "DateSchema")
		}
	}

	// Test mode checks
	if !registry.IsSingleFileMode() {
		t.Error("Should be in single file mode")
	}
	if registry.GetSingleFileName() != "custom-schemas.ts" {
		t.Errorf("GetSingleFileName() = %v, want %v", registry.GetSingleFileName(), "custom-schemas.ts")
	}
}

func TestCustomTypeRegistry_LoadFromConfig_InvalidMode(t *testing.T) {
	registry := NewCustomTypeRegistry()
	tempDir := testutils.TempDir(t)

	configContent := `typescript-zod:
  output:
    mode: "invalid-mode"`

	configPath := testutils.WriteFile(t, tempDir, "invalid-config.yaml", configContent)

	err := registry.LoadFromConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid output mode")
	}
	if !contains(err.Error(), "invalid output mode") {
		t.Errorf("Error should mention invalid output mode, got: %v", err)
	}
}

func TestCustomTypeRegistry_SaveExampleConfig(t *testing.T) {
	registry := NewCustomTypeRegistry()
	tempDir := testutils.TempDir(t)
	configPath := testutils.WriteFile(t, tempDir, "example.yaml", "")

	err := registry.SaveExampleConfig(configPath)
	if err != nil {
		t.Fatalf("SaveExampleConfig failed: %v", err)
	}

	// Check that file was created and contains expected content
	testutils.AssertFileExists(t, configPath)
	content := testutils.ReadFile(t, configPath)

	expectedContent := []string{
		"typescript-zod:",
		"output:",
		"customTypes:",
		"date-time:",
		"zodType:",
		"email:",
	}

	for _, expected := range expectedContent {
		if !contains(content, expected) {
			t.Errorf("Example config should contain '%s'", expected)
		}
	}
}

// Helper function since strings.Contains might not be available in all test environments
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}())
}
