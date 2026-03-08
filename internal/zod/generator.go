package zod

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"dtoForge/internal/generator"
)

// ZodGenerator implements the Generator interface for TypeScript/Zod
type ZodGenerator struct {
	customTypes *CustomTypeRegistry
	selfName    string // Name of the current self-recursive DTO being generated (empty if not recursive)
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
			if dto.IsSelfRecursive {
				g.selfName = dto.Name
			} else {
				g.selfName = ""
			}
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
		DTOs                   []generator.DTO
		Config                 generator.Config
		Imports                []string
		PackageName            string
		GenerateHelpers        bool
		GenerateSchemaRegistry bool
		GenerateSchemaNames    bool
	}{
		DTOs:                   dtos,
		Config:                 config,
		Imports:                []string{}, // Not using for now since we have import in template
		PackageName:            g.getPackageName(config),
		GenerateHelpers:        genConfig.GenerateHelpers,
		GenerateSchemaRegistry: genConfig.GenerateSchemaRegistry,
		GenerateSchemaNames:    genConfig.GenerateSchemaNames,
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
		DTOs                   []generator.DTO
		Config                 generator.Config
		PackageName            string
		GenerateHelpers        bool
		GenerateSchemaRegistry bool
		GenerateSchemaNames    bool
	}{
		DTOs:                   dtos,
		Config:                 config,
		PackageName:            g.getPackageName(config),
		GenerateHelpers:        genConfig.GenerateHelpers,
		GenerateSchemaRegistry: genConfig.GenerateSchemaRegistry,
		GenerateSchemaNames:    genConfig.GenerateSchemaNames,
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
		"toZodType":             g.toZodType,
		"toTSType":              g.toTSType,
		"propertyToZodType":     g.propertyToZodType,
		"isEffectivelyRequired": g.isEffectivelyRequired,
		"toCamelCase":           g.toCamelCase,
		"toPascalCase":          g.toPascalCase,
		"toKebabCase":           g.toKebabCase,
		"hasDescription":        g.hasDescription,
		"setCurrentDTO": func(name string, isSelfRecursive bool) string {
			if isSelfRecursive {
				g.selfName = name
			} else {
				g.selfName = ""
			}
			return ""
		},
		"len": func(slice []string) int { return len(slice) },
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"lt":  func(a, b int) bool { return a < b },
		"not": func(b bool) bool { return !b },
	}
}

func (g *ZodGenerator) getPackageName(config generator.Config) string {
	if config.PackageName != "" {
		return config.PackageName
	}
	return "generated-zod-schemas"
}

// sortDTOsByDependency sorts DTOs to handle dependencies correctly using topological sort
func (g *ZodGenerator) sortDTOsByDependency(dtos []generator.DTO) []generator.DTO {
	// Build dependency graph and perform topological sort
	dependencies := buildDependencyGraph(dtos)
	selfRecursive := detectAndRemoveSelfReferences(dependencies)
	sorted := topologicalSort(dtos, dependencies)

	// Mark self-recursive DTOs
	for i := range sorted {
		if selfRecursive[sorted[i].Name] {
			sorted[i].IsSelfRecursive = true
		}
	}

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
		if t.RefName == g.selfName {
			baseType = fmt.Sprintf("z.lazy(() => %sSchema)", t.RefName)
		} else {
			baseType = fmt.Sprintf("%sSchema", t.RefName)
		}
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
	case generator.NullType:
		baseType = "z.null()"
	case generator.MapType:
		valueType := g.toZodType(t.ValueType, false, false)
		baseType = fmt.Sprintf("z.record(z.string(), %s)", valueType)
	case generator.UnionType:
		// Handle oneOf/union types
		var unionTypes []string
		for _, unionType := range t.Types {
			unionTypes = append(unionTypes, g.toZodType(unionType, false, false))
		}
		if len(unionTypes) > 0 {
			baseType = fmt.Sprintf("z.union([%s])", strings.Join(unionTypes, ", "))
		} else {
			baseType = "z.unknown()"
		}
	case generator.IntersectionType:
		// Handle allOf/intersection types (schema composition)
		var intersectionTypes []string
		for _, intersectionType := range t.Types {
			intersectionTypes = append(intersectionTypes, g.toZodType(intersectionType, false, false))
		}
		if len(intersectionTypes) >= 2 {
			// Zod's intersection() only takes 2 args, so chain for 3+
			baseType = fmt.Sprintf("z.intersection(%s, %s)", intersectionTypes[0], intersectionTypes[1])
			for i := 2; i < len(intersectionTypes); i++ {
				baseType = fmt.Sprintf("z.intersection(%s, %s)", baseType, intersectionTypes[i])
			}
		} else if len(intersectionTypes) == 1 {
			baseType = intersectionTypes[0]
		} else {
			baseType = "z.unknown()"
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

// toTSType converts an IRType to a plain TypeScript type string (for manual interface declarations)
func (g *ZodGenerator) toTSType(irType generator.IRType, nullable bool) string {
	var baseType string

	switch t := irType.(type) {
	case generator.PrimitiveType:
		switch t.Name {
		case "string":
			baseType = "string"
		case "number", "integer":
			baseType = "number"
		case "boolean":
			baseType = "boolean"
		default:
			baseType = "unknown"
		}
	case generator.ArrayType:
		elementType := g.toTSType(t.ElementType, false)
		baseType = fmt.Sprintf("readonly %s[]", elementType)
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
	case generator.MapType:
		valueType := g.toTSType(t.ValueType, false)
		baseType = fmt.Sprintf("Record<string, %s>", valueType)
	case generator.UnionType:
		var unionTypes []string
		for _, unionType := range t.Types {
			unionTypes = append(unionTypes, g.toTSType(unionType, false))
		}
		if len(unionTypes) > 0 {
			baseType = fmt.Sprintf("(%s)", strings.Join(unionTypes, " | "))
		} else {
			baseType = "unknown"
		}
	case generator.IntersectionType:
		var intersectionTypes []string
		for _, intersectionType := range t.Types {
			intersectionTypes = append(intersectionTypes, g.toTSType(intersectionType, false))
		}
		if len(intersectionTypes) > 0 {
			baseType = fmt.Sprintf("(%s)", strings.Join(intersectionTypes, " & "))
		} else {
			baseType = "unknown"
		}
	case generator.NullType:
		baseType = "null"
	default:
		baseType = "unknown"
	}

	if nullable {
		return fmt.Sprintf("%s | null", baseType)
	}

	return baseType
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

// X-NULLABLE SUPPORT

// hasXNullable checks if a property has x-nullable: true extension
func (g *ZodGenerator) hasXNullable(prop generator.Property) bool {
	// Check if x-nullable handling is enabled
	if !g.customTypes.IsXNullableEnabled() {
		return false
	}

	if prop.Extensions == nil {
		return false
	}
	if xNullable, ok := prop.Extensions["x-nullable"]; ok {
		// Handle both bool and string representations
		switch v := xNullable.(type) {
		case bool:
			return v
		case string:
			return v == "true"
		}
	}
	return false
}

// XNullableEffect represents the effect of x-nullable on a property
type XNullableEffect struct {
	MakeNullable bool // Add .nullable() to the type
	MakeOptional bool // Add .optional() to the type
}

// getXNullableEffect determines the effect of x-nullable based on config
func (g *ZodGenerator) getXNullableEffect(prop generator.Property) XNullableEffect {
	if !g.hasXNullable(prop) {
		return XNullableEffect{MakeNullable: false, MakeOptional: false}
	}

	behavior := g.customTypes.GetXNullableBehavior()
	switch behavior {
	case XNullableBehaviorNull:
		return XNullableEffect{MakeNullable: true, MakeOptional: false}
	case XNullableBehaviorUndefined:
		return XNullableEffect{MakeNullable: false, MakeOptional: true}
	case XNullableBehaviorNullish:
		return XNullableEffect{MakeNullable: true, MakeOptional: true}
	default:
		return XNullableEffect{MakeNullable: true, MakeOptional: false} // Default to "null"
	}
}

// isEffectivelyRequired checks if a property is effectively required after considering x-nullable
func (g *ZodGenerator) isEffectivelyRequired(prop generator.Property) bool {
	if !prop.Required {
		return false
	}
	effect := g.getXNullableEffect(prop)
	return !effect.MakeOptional
}

// propertyToZodType converts a property to Zod type considering x-nullable
func (g *ZodGenerator) propertyToZodType(prop generator.Property) string {
	effect := g.getXNullableEffect(prop)
	nullable := prop.Nullable || effect.MakeNullable
	optional := !prop.Required || effect.MakeOptional
	return g.toZodType(prop.Type, nullable, optional)
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
