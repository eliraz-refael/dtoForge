package typescript

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

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

	// Sort DTOs to ensure consistent output and handle dependencies
	sortedDTOs := g.sortDTOsByDependency(dtos)

	// Get generation settings
	genConfig := g.customTypes.GetGenerationConfig()

	// Generate based on output mode
	if g.customTypes.IsSingleFileMode() {
		if err := g.generateSingleFile(sortedDTOs, config, genConfig); err != nil {
			return fmt.Errorf("failed to generate single file: %w", err)
		}
	} else {
		// Generate index file that exports all schemas
		if err := g.generateIndexFile(sortedDTOs, config, genConfig); err != nil {
			return fmt.Errorf("failed to generate index file: %w", err)
		}

		// Generate individual files for each DTO
		for _, dto := range sortedDTOs {
			if err := g.generateDTOFile(dto, config, genConfig); err != nil {
				return fmt.Errorf("failed to generate file for DTO %s: %w", dto.Name, err)
			}
		}
	}

	// Generate package.json if needed
	if genConfig.GeneratePackageJson {
		if err := g.generatePackageJSON(config); err != nil {
			return fmt.Errorf("failed to generate package.json: %w", err)
		}
	}

	return nil
}

// generateSingleFile creates a single TypeScript file with all DTOs
func (g *TypeScriptGenerator) generateSingleFile(dtos []generator.DTO, config generator.Config, genConfig GenerationConfig) error {
	filename := g.customTypes.GetSingleFileName()
	filepath := filepath.Join(config.OutputFolder, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	tmpl, err := template.New("single-file").Funcs(g.templateFuncs()).Parse(singleFileTemplate)
	if err != nil {
		return err
	}

	// Calculate all imports needed for all DTOs
	allFormats := []string{}
	for _, dto := range dtos {
		dtoFormats := g.getUsedFormatsInDTO(dto)
		for _, format := range dtoFormats {
			// Add to allFormats if not already present
			found := false
			for _, existing := range allFormats {
				if existing == format {
					found = true
					break
				}
			}
			if !found {
				allFormats = append(allFormats, format)
			}
		}
	}
	allImports := g.customTypes.GetAllImports(allFormats)

	data := struct {
		DTOs                  []generator.DTO
		Config                generator.Config
		Imports               []string
		PackageName           string
		GeneratePartialCodecs bool
		GenerateHelpers       bool
	}{
		DTOs:                  dtos,
		Config:                config,
		Imports:               allImports,
		PackageName:           g.getPackageName(config),
		GeneratePartialCodecs: genConfig.GeneratePartialCodecs,
		GenerateHelpers:       genConfig.GenerateHelpers,
	}

	return tmpl.Execute(file, data)
}

func (g *TypeScriptGenerator) generateDTOFile(dto generator.DTO, config generator.Config, genConfig GenerationConfig) error {
	filename := fmt.Sprintf("%s%s", g.toKebabCase(dto.Name), g.FileExtension())
	filepath := filepath.Join(config.OutputFolder, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	tmpl, err := template.New("dto").Funcs(g.templateFuncs()).Parse(dtoTemplate)
	if err != nil {
		return err
	}

	data := struct {
		DTO                   generator.DTO
		Config                generator.Config
		Imports               []string
		PackageName           string
		GeneratePartialCodecs bool
	}{
		DTO:                   dto,
		Config:                config,
		Imports:               g.calculateImports(dto),
		PackageName:           g.getPackageName(config),
		GeneratePartialCodecs: genConfig.GeneratePartialCodecs,
	}
	return tmpl.Execute(file, data)
}

// Updated generateIndexFile to accept genConfig
func (g *TypeScriptGenerator) generateIndexFile(dtos []generator.DTO, config generator.Config, genConfig GenerationConfig) error {
	filepath := filepath.Join(config.OutputFolder, "index.ts")

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	tmpl, err := template.New("index").Funcs(g.templateFuncs()).Parse(indexTemplate)
	if err != nil {
		return err
	}

	data := struct {
		DTOs            []generator.DTO
		Config          generator.Config
		PackageName     string
		GenerateHelpers bool
	}{
		DTOs:            dtos,
		Config:          config,
		PackageName:     g.getPackageName(config),
		GenerateHelpers: genConfig.GenerateHelpers,
	}

	return tmpl.Execute(file, data)
}

// generatePackageJSON creates a package.json for the generated code
func (g *TypeScriptGenerator) generatePackageJSON(config generator.Config) error {
	filepath := filepath.Join(config.OutputFolder, "package.json")

	// Don't overwrite existing package.json
	if _, err := os.Stat(filepath); err == nil {
		return nil
	}

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	tmpl, err := template.New("package").Funcs(g.templateFuncs()).Parse(packageJSONTemplate)
	if err != nil {
		return err
	}

	data := struct {
		PackageName string
	}{
		PackageName: g.getPackageName(config),
	}

	return tmpl.Execute(file, data)
}

// Helper functions for templates
func (g *TypeScriptGenerator) templateFuncs() template.FuncMap {
	return template.FuncMap{
		"toIoTsType":     g.toIoTsType,
		"toTSType":       g.toTSType,
		"toCamelCase":    g.toCamelCase,
		"toPascalCase":   g.toPascalCase,
		"toKebabCase":    g.toKebabCase,
		"isRequired":     g.isRequired,
		"hasDescription": g.hasDescription,
		"join":           strings.Join,
		"quote":          g.quote,
		"len":            func(slice []string) int { return len(slice) },
		"add":            func(a, b int) int { return a + b },
	}
}

// toIoTsType converts an IRType to io-ts codec using custom type mappings
func (g *TypeScriptGenerator) toIoTsType(irType generator.IRType, nullable bool) string {
	var baseType string

	switch t := irType.(type) {
	case generator.PrimitiveType:
		switch t.Name {
		case "string":
			// Check for custom format mapping
			if t.Format != "" {
				if mapping, exists := g.customTypes.Get(t.Format); exists {
					baseType = mapping.IoTsType
				} else {
					baseType = "t.string"
				}
			} else {
				baseType = "t.string"
			}
		case "number", "integer":
			baseType = "t.number"
		case "boolean":
			baseType = "t.boolean"
		default:
			baseType = "t.unknown"
		}
	case generator.ArrayType:
		elementType := g.toIoTsType(t.ElementType, false)
		baseType = fmt.Sprintf("t.array(%s)", elementType)
	case generator.ReferenceType:
		baseType = fmt.Sprintf("%sCodec", t.RefName)
	case generator.EnumType:
		values := make([]string, len(t.Values))
		for i, v := range t.Values {
			values[i] = fmt.Sprintf("'%s': null", v)
		}
		baseType = fmt.Sprintf("t.keyof({%s})", strings.Join(values, ", "))
	case generator.ObjectType:
		if t.RefName != "" {
			baseType = fmt.Sprintf("%sCodec", t.RefName)
		} else {
			baseType = "t.unknown" // inline objects need special handling
		}
	default:
		baseType = "t.unknown"
	}

	if nullable {
		return fmt.Sprintf("t.union([%s, t.null])", baseType)
	}

	return baseType
}

// toTSType converts an IRType to TypeScript type using custom type mappings
func (g *TypeScriptGenerator) toTSType(irType generator.IRType, nullable bool) string {
	var baseType string

	switch t := irType.(type) {
	case generator.PrimitiveType:
		switch t.Name {
		case "string":
			// Check for custom format mapping
			if t.Format != "" {
				if mapping, exists := g.customTypes.Get(t.Format); exists {
					baseType = mapping.TypeScriptType
				} else {
					baseType = "string"
				}
			} else {
				baseType = "string"
			}
		case "number", "integer":
			baseType = "number"
		case "boolean":
			baseType = "boolean"
		default:
			baseType = "unknown"
		}
	case generator.ArrayType:
		elementType := g.toTSType(t.ElementType, false)
		baseType = fmt.Sprintf("%s[]", elementType)
	case generator.ReferenceType:
		baseType = t.RefName
	case generator.EnumType:
		values := make([]string, len(t.Values))
		for i, v := range t.Values {
			values[i] = fmt.Sprintf("'%s'", v)
		}
		baseType = strings.Join(values, " | ")
	case generator.ObjectType:
		if t.RefName != "" {
			baseType = t.RefName
		} else {
			baseType = "Record<string, unknown>"
		}
	default:
		baseType = "unknown"
	}

	if nullable {
		return fmt.Sprintf("%s | null", baseType)
	}

	return baseType
}

// Utility functions (same as before)
func (g *TypeScriptGenerator) toCamelCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func (g *TypeScriptGenerator) toPascalCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (g *TypeScriptGenerator) toKebabCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result.WriteRune('-')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func (g *TypeScriptGenerator) isRequired(propName string, required []string) bool {
	for _, req := range required {
		if req == propName {
			return true
		}
	}
	return false
}

func (g *TypeScriptGenerator) hasDescription(desc string) bool {
	return strings.TrimSpace(desc) != ""
}

func (g *TypeScriptGenerator) quote(s string) string {
	return fmt.Sprintf("'%s'", s)
}

func (g *TypeScriptGenerator) getPackageName(config generator.Config) string {
	if config.PackageName != "" {
		return config.PackageName
	}
	return "generated-schemas"
}

// calculateImports determines what needs to be imported for a DTO using custom types
func (g *TypeScriptGenerator) calculateImports(dto generator.DTO) []string {
	// Get all formats used in this DTO
	usedFormats := g.getUsedFormatsInDTO(dto)

	// Use the custom type registry to get the appropriate imports
	return g.customTypes.GetAllImports(usedFormats)
}

// getUsedFormatsInDTO finds all formats used in a single DTO
func (g *TypeScriptGenerator) getUsedFormatsInDTO(dto generator.DTO) []string {
	formatSet := make(map[string]bool)
	var formats []string

	for _, prop := range dto.Properties {
		if prim, ok := prop.Type.(generator.PrimitiveType); ok {
			if prim.Format != "" && !formatSet[prim.Format] {
				formats = append(formats, prim.Format)
				formatSet[prim.Format] = true
			}
		}
	}

	return formats
}

// sortDTOsByDependency sorts DTOs to handle dependencies correctly
func (g *TypeScriptGenerator) sortDTOsByDependency(dtos []generator.DTO) []generator.DTO {
	// Simple alphabetical sort for now - could be enhanced with proper dependency resolution
	sorted := make([]generator.DTO, len(dtos))
	copy(sorted, dtos)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	return sorted
}
