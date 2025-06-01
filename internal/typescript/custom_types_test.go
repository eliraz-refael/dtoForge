package typescript

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
			format: "date-time",
			expected: CustomTypeMapping{
				IoTsType:        "DateFromISOString",
				TypeScriptType:  "Date",
				ImportStatement: "import { DateFromISOString } from 'io-ts-types';",
			},
		},
		{
			format: "uuid",
			expected: CustomTypeMapping{
				IoTsType:        "t.string",
				TypeScriptType:  "string",
				ImportStatement: "",
			},
		},
		{
			format: "email",
			expected: CustomTypeMapping{
				IoTsType:        "t.string",
				TypeScriptType:  "string",
				ImportStatement: "",
			},
		},
		{
			format: "uri",
			expected: CustomTypeMapping{
				IoTsType:        "t.string",
				TypeScriptType:  "string",
				ImportStatement: "",
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

			if mapping.IoTsType != tt.expected.IoTsType {
				t.Errorf("IoTsType = %v, want %v", mapping.IoTsType, tt.expected.IoTsType)
			}
			if mapping.TypeScriptType != tt.expected.TypeScriptType {
				t.Errorf("TypeScriptType = %v, want %v", mapping.TypeScriptType, tt.expected.TypeScriptType)
			}
			if mapping.ImportStatement != tt.expected.ImportStatement {
				t.Errorf("ImportStatement = %v, want %v", mapping.ImportStatement, tt.expected.ImportStatement)
			}
		})
	}
}

func TestCustomTypeRegistry_Register(t *testing.T) {
	registry := NewCustomTypeRegistry()

	customMapping := CustomTypeMapping{
		IoTsType:        "CustomType",
		TypeScriptType:  "Custom",
		ImportStatement: "import { CustomType } from './custom';",
	}

	registry.Register("custom-format", customMapping)

	mapping, exists := registry.Get("custom-format")
	if !exists {
		t.Error("Expected custom mapping to exist after registration")
		return
	}

	if mapping.IoTsType != customMapping.IoTsType {
		t.Errorf("IoTsType = %v, want %v", mapping.IoTsType, customMapping.IoTsType)
	}
}

func TestCustomTypeRegistry_GetAllImports(t *testing.T) {
	registry := NewCustomTypeRegistry()

	// Add some custom mappings with imports
	registry.Register("custom1", CustomTypeMapping{
		IoTsType:        "Custom1",
		ImportStatement: "import { Custom1 } from './custom1';",
	})
	registry.Register("custom2", CustomTypeMapping{
		IoTsType:        "Custom2",
		ImportStatement: "import { Custom2 } from './custom2';",
	})
	registry.Register("custom3", CustomTypeMapping{
		IoTsType:        "Custom3",
		ImportStatement: "", // No import
	})

	usedFormats := []string{"custom1", "custom2", "custom3", "email"} // email has no import
	imports := registry.GetAllImports(usedFormats)

	// Should always include io-ts import first
	if len(imports) < 1 || imports[0] != "import * as t from 'io-ts';" {
		t.Error("First import should always be io-ts")
	}

	// Should include custom imports (sorted alphabetically)
	expectedImports := []string{
		"import * as t from 'io-ts';",
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
	if !config.GeneratePartialCodecs {
		t.Error("Should generate partial codecs by default")
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

	// Create a test config file with the correct structure for TypeScript generator
	configContent := `output:
  folder: "./custom-output"
  mode: "single"
  singleFileName: "custom-schemas.ts"
generation:
  generatePackageJson: false
  generatePartialCodecs: false
  generateHelpers: false
customTypes:
  uuid:
    ioTsType: "UUID"
    typeScriptType: "UUID"
    import: "import { UUID } from './uuid';"
  custom-date:
    ioTsType: "DateCodec"
    typeScriptType: "CustomDate"
    import: "import { DateCodec } from './dates';"`

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
	if genConfig.GeneratePartialCodecs {
		t.Error("GeneratePartialCodecs should be false")
	}
	if genConfig.GenerateHelpers {
		t.Error("GenerateHelpers should be false")
	}

	// Test custom types were loaded
	uuidMapping, exists := registry.Get("uuid")
	if !exists {
		t.Error("UUID mapping should exist")
	} else {
		if uuidMapping.IoTsType != "UUID" {
			t.Errorf("UUID IoTsType = %v, want %v", uuidMapping.IoTsType, "UUID")
		}
		if uuidMapping.ImportStatement != "import { UUID } from './uuid';" {
			t.Errorf("UUID ImportStatement = %v, want %v", uuidMapping.ImportStatement, "import { UUID } from './uuid';")
		}
	}

	customDateMapping, exists := registry.Get("custom-date")
	if !exists {
		t.Error("Custom date mapping should exist")
	} else {
		if customDateMapping.IoTsType != "DateCodec" {
			t.Errorf("CustomDate IoTsType = %v, want %v", customDateMapping.IoTsType, "DateCodec")
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

	configContent := `output:
  mode: "invalid-mode"`

	configPath := testutils.WriteFile(t, tempDir, "invalid-config.yaml", configContent)

	err := registry.LoadFromConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid output mode")
		return
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
		"output:",
		"customTypes:",
		"date-time:",
		"ioTsType:",
		"email:",
		"generation:",
	}

	for _, expected := range expectedContent {
		if !contains(content, expected) {
			t.Errorf("Example config should contain '%s'", expected)
		}
	}
}

// Helper function since strings.Contains might not be available in all test environments
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
