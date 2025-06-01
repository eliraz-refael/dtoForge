package zod

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"dtoForge/internal/generator"
)

// ZodGenerator implements the Generator interface for TypeScript/Zod
type ZodGenerator struct {
	customTypes *CustomTypeRegistry
}

// NewZodGenerator creates a new Zod generator
func NewZodGenerator() *ZodGenerator {
	return &ZodGenerator{}
}

// Language returns the language name
func (g *ZodGenerator) Language() string {
	return "typescript-zod"
}

// FileExtension returns the file extension for generated files
func (g *ZodGenerator) FileExtension() string {
	return ".ts"
}

// Generate creates TypeScript/Zod files from DTOs
func (g *ZodGenerator) Generate(dtos []generator.DTO, config generator.Config) error {
	// Initialize custom type registry
	g.customTypes = NewCustomTypeRegistry()

	// Load custom config if specified
	if config.ConfigFile != "" {
		if err := g.customTypes.LoadFromConfig(config.ConfigFile); err != nil {
			return fmt.Errorf("failed to load custom types config from %s: %w", config.ConfigFile, err)
		}
	}

	// Sort DTOs for consistent output
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

// generateDTOFile creates individual DTO files with Zod schemas
func (g *ZodGenerator) generateDTOFile(dto generator.DTO, config generator.Config, genConfig GenerationConfig) error {
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
		DTO         generator.DTO
		Config      generator.Config
		Imports     []string
		PackageName string
	}{
		DTO:         dto,
		Config:      config,
		Imports:     g.calculateImports(dto),
		PackageName: g.getPackageName(config),
	}

	return tmpl.Execute(file, data)
}

// generateSingleFile creates a single TypeScript file with all DTOs
func (g *ZodGenerator) generateSingleFile(dtos []generator.DTO, config generator.Config, genConfig GenerationConfig) error {
	filename := g.customTypes.GetSingleFileName()
	filepath := filepath.Join(config.OutputFolder, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	tmpl, err := template.New("single-file").Funcs(g.templateFuncs()).Parse(singleFileTemplate)
	if err != nil {
		return fmt.Errorf("template parse error: %w", err)
	}

	data := struct {
		DTOs            []generator.DTO
		Config          generator.Config
		Imports         []string
		PackageName     string
		GenerateHelpers bool
	}{
		DTOs:            dtos,
		Config:          config,
		Imports:         []string{}, // Not using for now since we have import in template
		PackageName:     g.getPackageName(config),
		GenerateHelpers: genConfig.GenerateHelpers,
	}

	err = tmpl.Execute(file, data)
	if err != nil {
		return fmt.Errorf("template execute error: %w", err)
	}

	return nil
}

// generateIndexFile creates the main index file that exports everything
func (g *ZodGenerator) generateIndexFile(dtos []generator.DTO, config generator.Config, genConfig GenerationConfig) error {
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
func (g *ZodGenerator) generatePackageJSON(config generator.Config) error {
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
func (g *ZodGenerator) templateFuncs() template.FuncMap {
	return template.FuncMap{
		"toZodType":      g.toZodType,
		"toCamelCase":    g.toCamelCase,
		"toPascalCase":   g.toPascalCase,
		"toKebabCase":    g.toKebabCase,
		"hasDescription": g.hasDescription,
		"len":            func(slice []string) int { return len(slice) },
		"add":            func(a, b int) int { return a + b },
		"sub":            func(a, b int) int { return a - b },
		"lt":             func(a, b int) bool { return a < b },
		"not":            func(b bool) bool { return !b },
	}
}

func (g *ZodGenerator) getPackageName(config generator.Config) string {
	if config.PackageName != "" {
		return config.PackageName
	}
	return "generated-zod-schemas"
}

// sortDTOsByDependency sorts DTOs to handle dependencies correctly
func (g *ZodGenerator) sortDTOsByDependency(dtos []generator.DTO) []generator.DTO {
	// Simple alphabetical sort for now - could be enhanced with proper dependency resolution
	sorted := make([]generator.DTO, len(dtos))
	copy(sorted, dtos)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	return sorted
}

// TYPE CONVERSION FUNCTIONS

// toZodType converts an IRType to Zod schema syntax
func (g *ZodGenerator) toZodType(irType generator.IRType, nullable bool, optional bool) string {
	var baseType string

	switch t := irType.(type) {
	case generator.PrimitiveType:
		baseType = g.primitiveToZod(t)
	case generator.ArrayType:
		elementType := g.toZodType(t.ElementType, false, false)
		baseType = fmt.Sprintf("z.array(%s)", elementType)
	case generator.ReferenceType:
		baseType = fmt.Sprintf("%sSchema", t.RefName)
	case generator.EnumType:
		values := make([]string, len(t.Values))
		for i, v := range t.Values {
			values[i] = fmt.Sprintf("'%s'", v)
		}
		baseType = fmt.Sprintf("z.enum([%s])", strings.Join(values, ", "))
	case generator.ObjectType:
		if t.RefName != "" {
			baseType = fmt.Sprintf("%sSchema", t.RefName)
		} else {
			baseType = "z.record(z.unknown())" // inline objects
		}
	default:
		baseType = "z.unknown()"
	}

	// Apply modifiers based on nullable and optional
	if nullable {
		baseType = fmt.Sprintf("%s.nullable()", baseType)
	}

	if optional {
		baseType = fmt.Sprintf("%s.optional()", baseType)
	}

	return baseType
}

// primitiveToZod converts primitive types to Zod equivalents
func (g *ZodGenerator) primitiveToZod(prim generator.PrimitiveType) string {
	switch prim.Name {
	case "string":
		return g.stringWithFormat(prim.Format)
	case "number", "integer":
		return "z.number()"
	case "boolean":
		return "z.boolean()"
	default:
		return "z.unknown()"
	}
}

// stringWithFormat applies Zod string validations based on OpenAPI format
func (g *ZodGenerator) stringWithFormat(format string) string {
	// Check for custom format mapping first
	if g.customTypes != nil {
		if mapping, exists := g.customTypes.Get(format); exists {
			return mapping.ZodType
		}
	}

	// Fall back to built-in Zod formats
	switch format {
	case "email":
		return "z.string().email()"
	case "uuid":
		return "z.string().uuid()"
	case "uri", "url":
		return "z.string().url()"
	case "date-time":
		return "z.string().datetime()"
	case "date":
		return "z.string().date()"
	case "":
		return "z.string()"
	default:
		// Unknown format, just use string with a comment
		return fmt.Sprintf("z.string() /* format: %s */", format)
	}
}

// UTILITY FUNCTIONS

func (g *ZodGenerator) toCamelCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func (g *ZodGenerator) toPascalCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (g *ZodGenerator) toKebabCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result.WriteRune('-')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func (g *ZodGenerator) hasDescription(desc string) bool {
	return strings.TrimSpace(desc) != ""
}

// calculateImports determines what needs to be imported for a DTO using custom types
func (g *ZodGenerator) calculateImports(dto generator.DTO) []string {
	// Get all formats used in this DTO
	usedFormats := g.getUsedFormatsInDTO(dto)

	// Use the custom type registry to get the appropriate imports
	return g.customTypes.GetAllImports(usedFormats)
}

// getUsedFormatsInDTO finds all formats used in a single DTO
func (g *ZodGenerator) getUsedFormatsInDTO(dto generator.DTO) []string {
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
