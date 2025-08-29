package typescript

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"dtoForge/internal/generator"
)

// SingleFileGenerator handles generation of all DTOs in a single TypeScript file
type SingleFileGenerator struct {
	customTypes *CustomTypeRegistry
}

// NewSingleFileGenerator creates a new single-file generator
func NewSingleFileGenerator(customTypes *CustomTypeRegistry) *SingleFileGenerator {
	return &SingleFileGenerator{
		customTypes: customTypes,
	}
}

// Generate creates a single TypeScript file containing all DTOs
func (g *SingleFileGenerator) Generate(dtos []generator.DTO, config generator.Config) error {
	// Sort DTOs by dependency to avoid "used before declaration" errors
	sortedDTOs := g.sortByDependency(dtos)

	// Prepare DTOs with template-specific data
	templateDTOs := g.prepareDTOs(sortedDTOs)

	// Calculate all imports needed
	allImports := g.collectAllImports(sortedDTOs)

	// Get generation settings
	genConfig := g.customTypes.GetGenerationConfig()

	// Generate the file
	filename := g.customTypes.GetSingleFileName()
	filepath := filepath.Join(config.OutputFolder, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filepath, err)
	}
	defer file.Close()

	// Prepare template data
	data := struct {
		DTOs                   []templateDTO
		Config                 generator.Config
		Imports                []string
		PackageName            string
		GeneratePartialCodecs  bool
		GenerateHelpers        bool
		GenerateSchemaRegistry bool
		GenerateSchemaNames    bool
		UseInterfaces          bool
	}{
		DTOs:                   templateDTOs,
		Config:                 config,
		Imports:                allImports,
		PackageName:            getPackageName(config),
		GeneratePartialCodecs:  genConfig.GeneratePartialCodecs,
		GenerateHelpers:        genConfig.GenerateHelpers,
		GenerateSchemaRegistry: genConfig.GenerateSchemaRegistry,
		GenerateSchemaNames:    genConfig.GenerateSchemaNames,
		UseInterfaces:          genConfig.UseInterfaces,
	}

	// Execute template
	tmpl, err := template.New("single-file").Funcs(g.templateFuncs()).Parse(singleFileTemplate)
	if err != nil {
		return fmt.Errorf("template parse error: %w", err)
	}

	err = tmpl.Execute(file, data)
	if err != nil {
		return fmt.Errorf("template execute error: %w", err)
	}

	return nil
}

// sortByDependency uses topological sort to order DTOs correctly
func (g *SingleFileGenerator) sortByDependency(dtos []generator.DTO) []generator.DTO {
	// Build dependency graph
	dependencies := buildDependencyGraph(dtos)

	// Perform topological sort
	return topologicalSort(dtos, dependencies)
}

// prepareDTOs adds template-specific data to DTOs
func (g *SingleFileGenerator) prepareDTOs(dtos []generator.DTO) []templateDTO {
	var result []templateDTO

	for _, dto := range dtos {
		hasRequired := false
		for _, prop := range dto.Properties {
			if prop.Required {
				hasRequired = true
				break
			}
		}

		result = append(result, templateDTO{
			DTO:               dto,
			HasRequiredFields: hasRequired,
		})
	}

	return result
}

// collectAllImports gathers all imports needed for all DTOs
func (g *SingleFileGenerator) collectAllImports(dtos []generator.DTO) []string {
	allFormats := []string{}
	formatSet := make(map[string]bool)

	for _, dto := range dtos {
		dtoFormats := getUsedFormatsInDTO(dto)
		for _, format := range dtoFormats {
			if !formatSet[format] {
				allFormats = append(allFormats, format)
				formatSet[format] = true
			}
		}
	}

	return g.customTypes.GetAllImports(allFormats)
}

// templateFuncs returns the template function map
func (g *SingleFileGenerator) templateFuncs() template.FuncMap {
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

// templateDTO wraps a DTO with additional template-specific data
type templateDTO struct {
	generator.DTO
	HasRequiredFields bool
}
