package typescript

import (
	"testing"

	"dtoForge/internal/generator"
)

func TestExtractDependencies_WithIntersectionType(t *testing.T) {
	// Create a DTO that uses allOf (IntersectionType) to reference another schema
	// This simulates schemas like:
	//   FixedMethodProperties:
	//     allOf:
	//       - $ref: '#/components/schemas/MethodPropertiesBase'
	//       - type: object
	//         properties: ...
	dto := generator.DTO{
		Name: "FixedMethodProperties",
		Type: "allOf",
		IntersectionType: &generator.IntersectionType{
			Types: []generator.IRType{
				generator.ReferenceType{RefName: "MethodPropertiesBase"},
				generator.ObjectType{Inline: true},
			},
		},
	}

	deps := extractDependencies(dto)

	if len(deps) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(deps))
	}

	if deps[0] != "MethodPropertiesBase" {
		t.Errorf("expected dependency 'MethodPropertiesBase', got '%s'", deps[0])
	}
}

func TestExtractDependencies_WithProperties(t *testing.T) {
	// Test that regular properties still have their dependencies extracted
	dto := generator.DTO{
		Name: "MyDTO",
		Type: "object",
		Properties: []generator.Property{
			{
				Name: "ref",
				Type: generator.ReferenceType{RefName: "OtherSchema"},
			},
		},
	}

	deps := extractDependencies(dto)

	if len(deps) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(deps))
	}

	if deps[0] != "OtherSchema" {
		t.Errorf("expected dependency 'OtherSchema', got '%s'", deps[0])
	}
}

func TestExtractDependencies_WithBothPropertiesAndIntersection(t *testing.T) {
	// Test a complex case with both properties and intersection type
	dto := generator.DTO{
		Name: "ComplexDTO",
		Type: "allOf",
		IntersectionType: &generator.IntersectionType{
			Types: []generator.IRType{
				generator.ReferenceType{RefName: "BaseSchema"},
			},
		},
		Properties: []generator.Property{
			{
				Name: "extra",
				Type: generator.ReferenceType{RefName: "ExtraSchema"},
			},
		},
	}

	deps := extractDependencies(dto)

	if len(deps) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(deps))
	}

	// Check both dependencies are present
	found := make(map[string]bool)
	for _, dep := range deps {
		found[dep] = true
	}

	if !found["BaseSchema"] {
		t.Error("expected dependency 'BaseSchema' not found")
	}
	if !found["ExtraSchema"] {
		t.Error("expected dependency 'ExtraSchema' not found")
	}
}

func TestTopologicalSort_AllOfDependencies(t *testing.T) {
	// Create DTOs where alphabetical order would be wrong
	// ZChild depends on AParent via allOf
	parent := generator.DTO{
		Name: "AParent",
		Type: "object",
		Properties: []generator.Property{
			{Name: "id", Type: generator.PrimitiveType{Name: "string"}},
		},
	}

	child := generator.DTO{
		Name: "ZChild",
		Type: "allOf",
		IntersectionType: &generator.IntersectionType{
			Types: []generator.IRType{
				generator.ReferenceType{RefName: "AParent"},
				generator.ObjectType{Inline: true},
			},
		},
	}

	// Pass them in alphabetically wrong order (ZChild first)
	dtos := []generator.DTO{child, parent}

	// Build dependencies and sort
	deps := buildDependencyGraph(dtos)
	sorted := topologicalSort(dtos, deps)

	// AParent must come before ZChild
	var parentIdx, childIdx int
	for i, dto := range sorted {
		if dto.Name == "AParent" {
			parentIdx = i
		}
		if dto.Name == "ZChild" {
			childIdx = i
		}
	}

	if parentIdx >= childIdx {
		t.Errorf("AParent (idx %d) should come before ZChild (idx %d)", parentIdx, childIdx)
	}
}

func TestTopologicalSort_MultiLevelAllOfDependencies(t *testing.T) {
	// Create a chain: GrandChild -> Child -> Parent
	parent := generator.DTO{
		Name: "Parent",
		Type: "object",
		Properties: []generator.Property{
			{Name: "id", Type: generator.PrimitiveType{Name: "string"}},
		},
	}

	child := generator.DTO{
		Name: "Child",
		Type: "allOf",
		IntersectionType: &generator.IntersectionType{
			Types: []generator.IRType{
				generator.ReferenceType{RefName: "Parent"},
			},
		},
	}

	grandChild := generator.DTO{
		Name: "GrandChild",
		Type: "allOf",
		IntersectionType: &generator.IntersectionType{
			Types: []generator.IRType{
				generator.ReferenceType{RefName: "Child"},
			},
		},
	}

	// Pass them in reverse order
	dtos := []generator.DTO{grandChild, child, parent}

	deps := buildDependencyGraph(dtos)
	sorted := topologicalSort(dtos, deps)

	// Find indices
	indices := make(map[string]int)
	for i, dto := range sorted {
		indices[dto.Name] = i
	}

	// Parent < Child < GrandChild
	if indices["Parent"] >= indices["Child"] {
		t.Errorf("Parent (idx %d) should come before Child (idx %d)", indices["Parent"], indices["Child"])
	}
	if indices["Child"] >= indices["GrandChild"] {
		t.Errorf("Child (idx %d) should come before GrandChild (idx %d)", indices["Child"], indices["GrandChild"])
	}
}
