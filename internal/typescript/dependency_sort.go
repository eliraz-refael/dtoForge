package typescript

import (
	"sort"

	"dtoForge/internal/generator"
)

// buildDependencyGraph analyzes DTOs and builds a dependency graph
func buildDependencyGraph(dtos []generator.DTO) map[string][]string {
	dependencies := make(map[string][]string)

	for _, dto := range dtos {
		deps := extractDependencies(dto)
		dependencies[dto.Name] = deps
	}

	return dependencies
}

// extractDependencies finds all type references in a DTO
func extractDependencies(dto generator.DTO) []string {
	seen := make(map[string]bool)
	var deps []string

	// Check properties for type references
	for _, prop := range dto.Properties {
		extractTypeRefs(prop.Type, &deps, seen)
	}

	// Check IntersectionType for allOf dependencies
	// This is critical: schemas using allOf store their dependencies here, not in properties
	if dto.IntersectionType != nil {
		for _, intersectionMember := range dto.IntersectionType.Types {
			extractTypeRefs(intersectionMember, &deps, seen)
		}
	}

	return deps
}

// extractTypeRefs recursively extracts type references from an IRType
func extractTypeRefs(irType generator.IRType, deps *[]string, seen map[string]bool) {
	switch t := irType.(type) {
	case generator.ReferenceType:
		// Direct reference to another type
		if !seen[t.RefName] {
			*deps = append(*deps, t.RefName)
			seen[t.RefName] = true
		}
	case generator.ArrayType:
		// Array of another type
		extractTypeRefs(t.ElementType, deps, seen)
	case generator.ObjectType:
		// Object type might reference another schema
		if t.RefName != "" && !seen[t.RefName] {
			*deps = append(*deps, t.RefName)
			seen[t.RefName] = true
		}
		// For inline objects, check properties for type references
		// This is important for allOf with inline objects that have $ref properties
		if t.Inline && t.DTORef != nil {
			for _, prop := range t.DTORef.Properties {
				extractTypeRefs(prop.Type, deps, seen)
			}
		}
	case generator.UnionType:
		// Union type (oneOf/anyOf) - check all types in the union
		for _, unionMember := range t.Types {
			extractTypeRefs(unionMember, deps, seen)
		}
	case generator.IntersectionType:
		// Intersection type (allOf) - check all types in the intersection
		for _, intersectionMember := range t.Types {
			extractTypeRefs(intersectionMember, deps, seen)
		}
	case generator.MapType:
		// Map type (Record<string, T>) - check the value type
		extractTypeRefs(t.ValueType, deps, seen)
	}
	// PrimitiveType, EnumType, and NullType don't have dependencies
}

// detectAndRemoveSelfReferences finds DTOs that reference themselves and removes
// those self-edges from the dependency graph so they don't create false cycles.
func detectAndRemoveSelfReferences(dependencies map[string][]string) map[string]bool {
	selfRecursive := make(map[string]bool)
	for name, deps := range dependencies {
		var cleaned []string
		for _, dep := range deps {
			if dep == name {
				selfRecursive[name] = true
			} else {
				cleaned = append(cleaned, dep)
			}
		}
		dependencies[name] = cleaned
	}
	return selfRecursive
}

// topologicalSort performs Kahn's algorithm to sort DTOs by dependencies
func topologicalSort(dtos []generator.DTO, dependencies map[string][]string) []generator.DTO {
	// Create a map for quick DTO lookup
	dtoMap := make(map[string]generator.DTO)
	for _, dto := range dtos {
		dtoMap[dto.Name] = dto
	}

	// Build reverse dependency map (who depends on me?)
	reverseDeps := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize in-degree for all DTOs
	for _, dto := range dtos {
		inDegree[dto.Name] = 0
		reverseDeps[dto.Name] = []string{}
	}

	// Build reverse dependencies and count in-degrees
	for dtoName, deps := range dependencies {
		for _, dep := range deps {
			// dep is used by dtoName
			if _, exists := dtoMap[dep]; exists {
				reverseDeps[dep] = append(reverseDeps[dep], dtoName)
				inDegree[dtoName]++
			}
		}
	}

	// Find all DTOs with no dependencies (in-degree = 0)
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	// Sort the initial queue for deterministic output
	sort.Strings(queue)

	// Process the queue
	var sorted []generator.DTO
	processed := make(map[string]bool)

	for len(queue) > 0 {
		// Pop from queue
		current := queue[0]
		queue = queue[1:]

		// Skip if already processed (shouldn't happen but be safe)
		if processed[current] {
			continue
		}

		// Add to sorted output
		sorted = append(sorted, dtoMap[current])
		processed[current] = true

		// Reduce in-degree for all dependents
		for _, dependent := range reverseDeps[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 && !processed[dependent] {
				queue = append(queue, dependent)
			}
		}

		// Sort queue after adding new elements to ensure deterministic output
		sort.Strings(queue)
	}

	// Handle cycles: add any remaining DTOs in alphabetical order
	if len(sorted) < len(dtos) {
		var remaining []generator.DTO

		for _, dto := range dtos {
			if !processed[dto.Name] {
				remaining = append(remaining, dto)
			}
		}

		// Sort remaining DTOs alphabetically for consistent output
		sort.Slice(remaining, func(i, j int) bool {
			return remaining[i].Name < remaining[j].Name
		})

		sorted = append(sorted, remaining...)
	}

	return sorted
}

// detectCycles finds circular dependencies in the dependency graph
// Returns a list of DTOs involved in cycles
func detectCycles(dependencies map[string][]string) []string {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var cyclic []string

	var hasCycle func(node string) bool
	hasCycle = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, dep := range dependencies[node] {
			if !visited[dep] {
				if hasCycle(dep) {
					return true
				}
			} else if recStack[dep] {
				// Found a cycle
				cyclic = append(cyclic, node)
				return true
			}
		}

		recStack[node] = false
		return false
	}

	// Check each unvisited node
	for node := range dependencies {
		if !visited[node] {
			hasCycle(node)
		}
	}

	return cyclic
}
