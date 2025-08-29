package typescript

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/template"

	"dtoForge/internal/generator"
)

// MultiFileGenerator handles generation of separate TypeScript files for each DTO
type MultiFileGenerator struct {
	customTypes *CustomTypeRegistry
}

// NewMultiFileGenerator creates a new multi-file generator
func NewMultiFileGenerator(customTypes *CustomTypeRegistry) *MultiFileGenerator {
	return &MultiFileGenerator{
		customTypes: customTypes,
	}
}

// Generate creates separate TypeScript files for each DTO plus an index file
func (g *MultiFileGenerator) Generate(dtos []generator.DTO, config generator.Config) error {
	// Sort DTOs alphabetically for consistent output
	sortedDTOs := g.sortAlphabetically(dtos)

	// Get generation settings
	genConfig := g.customTypes.GetGenerationConfig()

	// Generate individual files for each DTO
	for _, dto := range sortedDTOs {
		if err := g.generateDTOFile(dto, config, genConfig); err != nil {
			return fmt.Errorf("failed to generate file for DTO %s: %w", dto.Name, err)
		}
	}

	// Generate index file that exports all schemas
	if err := g.generateIndexFile(sortedDTOs, config, genConfig); err != nil {
		return fmt.Errorf("failed to generate index file: %w", err)
	}

	// Generate package.json if needed
	if genConfig.GeneratePackageJson {
		if err := g.generatePackageJSON(config); err != nil {
			return fmt.Errorf("failed to generate package.json: %w", err)
		}
	}

	return nil
}

// sortAlphabetically sorts DTOs by name for consistent output
func (g *MultiFileGenerator) sortAlphabetically(dtos []generator.DTO) []generator.DTO {
	sorted := make([]generator.DTO, len(dtos))
	copy(sorted, dtos)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	return sorted
}

// generateDTOFile creates a single TypeScript file for one DTO
func (g *MultiFileGenerator) generateDTOFile(dto generator.DTO, config generator.Config, genConfig GenerationConfig) error {
	filename := fmt.Sprintf("%s%s", toKebabCase(dto.Name), ".ts")
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

	// Check if DTO has required fields
	hasRequired := false
	for _, prop := range dto.Properties {
		if prop.Required {
			hasRequired = true
			break
		}
	}

	data := struct {
		DTO                   generator.DTO
		Config                generator.Config
		Imports               []string
		PackageName           string
		GeneratePartialCodecs bool
		GenerateHelpers       bool
		UseInterfaces         bool
		HasRequiredFields     bool
	}{
		DTO:                   dto,
		Config:                config,
		Imports:               calculateImports(dto, g.customTypes),
		PackageName:           getPackageName(config),
		GeneratePartialCodecs: genConfig.GeneratePartialCodecs,
		GenerateHelpers:       genConfig.GenerateHelpers,
		UseInterfaces:         genConfig.UseInterfaces,
		HasRequiredFields:     hasRequired,
	}

	return tmpl.Execute(file, data)
}

// generateIndexFile creates the main index file that exports everything
func (g *MultiFileGenerator) generateIndexFile(dtos []generator.DTO, config generator.Config, genConfig GenerationConfig) error {
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
		PackageName:            getPackageName(config),
		GenerateHelpers:        genConfig.GenerateHelpers,
		GenerateSchemaRegistry: genConfig.GenerateSchemaRegistry,
		GenerateSchemaNames:    genConfig.GenerateSchemaNames,
	}

	return tmpl.Execute(file, data)
}

// generatePackageJSON creates a package.json for the generated code
func (g *MultiFileGenerator) generatePackageJSON(config generator.Config) error {
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
		PackageName: getPackageName(config),
	}

	return tmpl.Execute(file, data)
}

// templateFuncs returns the template function map
func (g *MultiFileGenerator) templateFuncs() template.FuncMap {
	return template.FuncMap{
		"toIoTsType": func(irType generator.IRType, nullable bool) string {
			return toIoTsType(irType, nullable, g.customTypes)
		},
		"toTSType": func(irType generator.IRType, nullable bool) string {
			return toTSType(irType, nullable, g.customTypes)
		},
		"toCamelCase":          toCamelCase,
		"toPascalCase":         toPascalCase,
		"toKebabCase":          toKebabCase,
		"isRequired":           isRequired,
		"hasDescription":       hasDescription,
		"quote":                quote,
		"sanitizePropertyName": sanitizePropertyName,
		"safePropertyName":     safePropertyName,
		"len":                  func(slice []string) int { return len(slice) },
		"add":                  func(a, b int) int { return a + b },
	}
}
