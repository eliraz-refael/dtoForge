package typescript

import (
	"fmt"

	"dtoForge/internal/generator"
)

// TypeScriptGenerator implements the Generator interface for TypeScript/io-ts
type TypeScriptGenerator struct {
	customTypes *CustomTypeRegistry
}

// NewTypeScriptGenerator creates a new TypeScript generator
func NewTypeScriptGenerator() *TypeScriptGenerator {
	return &TypeScriptGenerator{}
}

// Language returns the language name
func (g *TypeScriptGenerator) Language() string {
	return "typescript"
}

// FileExtension returns the file extension for generated files
func (g *TypeScriptGenerator) FileExtension() string {
	return ".ts"
}

// Generate creates TypeScript/io-ts files from DTOs
func (g *TypeScriptGenerator) Generate(dtos []generator.DTO, config generator.Config) error {
	// Initialize custom type registry
	g.customTypes = NewCustomTypeRegistry()

	// Load custom config if specified
	if config.ConfigFile != "" {
		if err := g.customTypes.LoadFromConfig(config.ConfigFile); err != nil {
			return fmt.Errorf("failed to load custom types config from %s: %w", config.ConfigFile, err)
		}
	}

	// Dispatch to appropriate generator based on output mode
	if g.customTypes.IsSingleFileMode() {
		generator := NewSingleFileGenerator(g.customTypes)
		return generator.Generate(dtos, config)
	} else {
		generator := NewMultiFileGenerator(g.customTypes)
		return generator.Generate(dtos, config)
	}
}
