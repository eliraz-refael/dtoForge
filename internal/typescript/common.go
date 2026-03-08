package typescript

import (
	"fmt"
	"strings"
	"unicode"

	"dtoForge/internal/generator"
)

// Common utility functions shared between single-file and multi-file generators

// sanitizePropertyName ensures property names are valid JavaScript identifiers
// If not valid, wraps them in quotes for object literal syntax
func sanitizePropertyName(name string) string {
	if isValidJSIdentifier(name) {
		return name
	}
	return fmt.Sprintf(`"%s"`, name)
}

// safePropertyName combines camelCase conversion with property name sanitization
func safePropertyName(name string) string {
	camelName := toCamelCase(name)
	return sanitizePropertyName(camelName)
}

// isValidJSIdentifier checks if a string is a valid JavaScript identifier
func isValidJSIdentifier(name string) bool {
	if len(name) == 0 {
		return false
	}

	// Check if it starts with a letter, underscore, or dollar sign
	first := rune(name[0])
	if !unicode.IsLetter(first) && first != '_' && first != '$' {
		return false
	}

	// Check rest of characters - letters, digits, underscore, dollar sign
	for _, r := range name[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '$' {
			return false
		}
	}

	// Check if it's a reserved keyword
	return !isJSReservedWord(name)
}

// isJSReservedWord checks if a name is a JavaScript reserved word
func isJSReservedWord(name string) bool {
	reserved := map[string]bool{
		"break": true, "case": true, "catch": true, "class": true, "const": true,
		"continue": true, "debugger": true, "default": true, "delete": true, "do": true,
		"else": true, "export": true, "extends": true, "finally": true, "for": true,
		"function": true, "if": true, "import": true, "in": true, "instanceof": true,
		"new": true, "return": true, "super": true, "switch": true, "this": true,
		"throw": true, "try": true, "typeof": true, "var": true, "void": true,
		"while": true, "with": true, "yield": true, "let": true, "static": true,
		"enum": true, "implements": true, "interface": true, "package": true,
		"private": true, "protected": true, "public": true, "await": true, "async": true,
	}
	return reserved[name]
}

// toIoTsType converts an IRType to io-ts codec using custom type mappings
func toIoTsType(irType generator.IRType, nullable bool, customTypes *CustomTypeRegistry) string {
	var baseType string
	defaultUnknown := customTypes.GetDefaultUnknownType()

	switch t := irType.(type) {
	case generator.PrimitiveType:
		switch t.Name {
		case "string":
			// Check for custom format mapping
			if t.Format != "" {
				if mapping, exists := customTypes.Get(t.Format); exists {
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
			baseType = defaultUnknown
		}
	case generator.ArrayType:
		elementType := toIoTsType(t.ElementType, false, customTypes)
		baseType = fmt.Sprintf("t.readonlyArray(%s)", elementType)
	case generator.ReferenceType:
		baseType = t.RefName
	case generator.EnumType:
		var enumType string
		if len(t.Values) == 1 {
			// Single value: use t.literal instead of t.keyof
			enumType = fmt.Sprintf("t.literal('%s')", t.Values[0])
		} else {
			// Multiple values: use t.keyof
			values := make([]string, len(t.Values))
			for i, v := range t.Values {
				values[i] = fmt.Sprintf("'%s': null", v)
			}
			enumType = fmt.Sprintf("t.keyof({%s})", strings.Join(values, ", "))
		}

		// Handle null union
		if t.ContainsNull {
			baseType = fmt.Sprintf("t.union([%s, t.null])", enumType)
		} else {
			baseType = enumType
		}
	case generator.ObjectType:
		if t.RefName != "" {
			baseType = t.RefName
		} else if t.DTORef != nil && len(t.DTORef.Properties) > 0 {
			// Inline object with properties - generate the structure directly
			var requiredProps []string
			var optionalProps []string

			for _, prop := range t.DTORef.Properties {
				propType := propertyToIoTsType(prop, customTypes)
				propLine := fmt.Sprintf("%s: %s", safePropertyName(prop.Name), propType)
				if prop.Required {
					requiredProps = append(requiredProps, propLine)
				} else {
					optionalProps = append(optionalProps, propLine)
				}
			}

			if len(requiredProps) > 0 && len(optionalProps) > 0 {
				baseType = fmt.Sprintf("t.intersection([t.type({%s}), t.partial({%s})])",
					strings.Join(requiredProps, ", "), strings.Join(optionalProps, ", "))
			} else if len(requiredProps) > 0 {
				baseType = fmt.Sprintf("t.type({%s})", strings.Join(requiredProps, ", "))
			} else if len(optionalProps) > 0 {
				baseType = fmt.Sprintf("t.partial({%s})", strings.Join(optionalProps, ", "))
			} else {
				baseType = "t.type({})"
			}
		} else {
			baseType = defaultUnknown // inline objects need special handling
		}
	case generator.NullType:
		baseType = "t.null"
	case generator.MapType:
		valueType := toIoTsType(t.ValueType, false, customTypes)
		baseType = fmt.Sprintf("t.record(t.string, %s)", valueType)
	case generator.UnionType:
		// Handle oneOf/union types
		var unionTypes []string
		for _, unionType := range t.Types {
			unionTypes = append(unionTypes, toIoTsType(unionType, false, customTypes))
		}
		if len(unionTypes) > 0 {
			baseType = fmt.Sprintf("t.union([%s])", strings.Join(unionTypes, ", "))
		} else {
			baseType = defaultUnknown
		}
	case generator.IntersectionType:
		// Handle allOf/intersection types (schema composition)
		var intersectionTypes []string
		for _, intersectionType := range t.Types {
			intersectionTypes = append(intersectionTypes, toIoTsType(intersectionType, false, customTypes))
		}
		if len(intersectionTypes) == 0 {
			baseType = defaultUnknown
		} else if len(intersectionTypes) == 1 {
			// io-ts intersection requires at least 2 elements
			// If only 1 element, just use it directly without wrapping
			baseType = intersectionTypes[0]
		} else {
			baseType = fmt.Sprintf("t.intersection([%s])", strings.Join(intersectionTypes, ", "))
		}
	default:
		baseType = defaultUnknown
	}

	// Don't double-wrap if enum already handles null
	if nullable {
		if enum, ok := irType.(generator.EnumType); ok && enum.ContainsNull {
			// Enum already handles null, don't wrap again
			return baseType
		}
		return fmt.Sprintf("t.union([%s, t.null])", baseType)
	}

	return baseType
}

// toTSType converts an IRType to TypeScript type using custom type mappings.
// When codecCompat is true, generates types matching io-ts codec output:
//   - Arrays use ReadonlyArray<T> (matching t.readonlyArray)
//   - Maps use { [key: string]: T } (avoiding Record<> which can be shadowed by local exports)
func toTSType(irType generator.IRType, nullable bool, customTypes *CustomTypeRegistry) string {
	return toTSTypeInternal(irType, nullable, false, customTypes)
}

func toCodecTSType(irType generator.IRType, nullable bool, customTypes *CustomTypeRegistry) string {
	return toTSTypeInternal(irType, nullable, true, customTypes)
}

func toTSTypeInternal(irType generator.IRType, nullable bool, codecCompat bool, customTypes *CustomTypeRegistry) string {
	var baseType string
	defaultUnknown := "unknown"

	switch t := irType.(type) {
	case generator.PrimitiveType:
		switch t.Name {
		case "string":
			if t.Format != "" {
				if mapping, exists := customTypes.Get(t.Format); exists {
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
			baseType = defaultUnknown
		}
	case generator.ArrayType:
		elementType := toTSTypeInternal(t.ElementType, false, codecCompat, customTypes)
		if codecCompat {
			baseType = fmt.Sprintf("ReadonlyArray<%s>", elementType)
		} else {
			baseType = fmt.Sprintf("%s[]", elementType)
		}
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
		} else if codecCompat {
			baseType = "{ [key: string]: unknown }"
		} else {
			baseType = "Record<string, unknown>"
		}
	case generator.MapType:
		valueType := toTSTypeInternal(t.ValueType, false, codecCompat, customTypes)
		if codecCompat {
			baseType = fmt.Sprintf("{ [key: string]: %s }", valueType)
		} else {
			baseType = fmt.Sprintf("Record<string, %s>", valueType)
		}
	case generator.UnionType:
		var unionTypes []string
		for _, unionType := range t.Types {
			unionTypes = append(unionTypes, toTSTypeInternal(unionType, false, codecCompat, customTypes))
			// If the union already contains a NullType, suppress the nullable flag
			// to avoid generating redundant "| null" (e.g. "(Foo | null) | null")
			if _, isNull := unionType.(generator.NullType); isNull {
				nullable = false
			}
		}
		if len(unionTypes) > 0 {
			baseType = fmt.Sprintf("(%s)", strings.Join(unionTypes, " | "))
		} else {
			baseType = defaultUnknown
		}
	case generator.IntersectionType:
		var intersectionTypes []string
		for _, intersectionType := range t.Types {
			intersectionTypes = append(intersectionTypes, toTSTypeInternal(intersectionType, false, codecCompat, customTypes))
		}
		if len(intersectionTypes) > 0 {
			baseType = fmt.Sprintf("(%s)", strings.Join(intersectionTypes, " & "))
		} else {
			baseType = defaultUnknown
		}
	case generator.NullType:
		baseType = "null"
	default:
		baseType = defaultUnknown
	}

	if nullable {
		return fmt.Sprintf("%s | null", baseType)
	}

	return baseType
}

// propertyToCodecTSType converts a property to a TypeScript type matching the codec.
// For optional properties (not effectively required), adds | null since t.partial() makes fields nullable.
func propertyToCodecTSType(prop generator.Property, isRequired bool, customTypes *CustomTypeRegistry) string {
	effect := getXNullableEffect(prop, customTypes)
	nullable := prop.Nullable || effect.MakeNullable

	// In t.partial(), fields become T | null even if not explicitly nullable
	if !isRequired {
		nullable = true
	}

	return toCodecTSType(prop.Type, nullable, customTypes)
}

// Case conversion utilities
func toCamelCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func toPascalCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func toKebabCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result.WriteRune('-')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// isRequired checks if a property name is in the required list
func isRequired(propName string, required []string) bool {
	for _, req := range required {
		if req == propName {
			return true
		}
	}
	return false
}

// hasDescription checks if a description is non-empty
func hasDescription(desc string) bool {
	return strings.TrimSpace(desc) != ""
}

// hasXNullable checks if a property has x-nullable: true extension
func hasXNullable(prop generator.Property, customTypes *CustomTypeRegistry) bool {
	// Check if x-nullable handling is enabled
	if !customTypes.IsXNullableEnabled() {
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
		default:
			// Unexpected type, return false
		}
	}
	return false
}

// XNullableEffect represents the effect of x-nullable on a property
type XNullableEffect struct {
	MakeNullable bool // Add null to the type union
	MakeOptional bool // Move property to partial (make it optional)
}

// getXNullableEffect determines the effect of x-nullable based on config
func getXNullableEffect(prop generator.Property, customTypes *CustomTypeRegistry) XNullableEffect {
	if !hasXNullable(prop, customTypes) {
		return XNullableEffect{MakeNullable: false, MakeOptional: false}
	}

	behavior := customTypes.GetXNullableBehavior()
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
// A property is NOT effectively required if:
// - It's not marked as required in OpenAPI
// - OR x-nullable makes it optional (behavior is "undefined" or "nullish")
func isEffectivelyRequired(prop generator.Property, customTypes *CustomTypeRegistry) bool {
	if !prop.Required {
		return false
	}
	effect := getXNullableEffect(prop, customTypes)
	return !effect.MakeOptional
}

// propertyToIoTsType converts a property to io-ts type considering x-nullable
func propertyToIoTsType(prop generator.Property, customTypes *CustomTypeRegistry) string {
	// Check if property should be nullable due to x-nullable extension or standard nullable
	effect := getXNullableEffect(prop, customTypes)
	nullable := prop.Nullable || effect.MakeNullable
	return toIoTsType(prop.Type, nullable, customTypes)
}

// quote wraps a string in single quotes
func quote(s string) string {
	return fmt.Sprintf("'%s'", s)
}

// calculateImports determines what needs to be imported for a DTO using custom types
func calculateImports(dto generator.DTO, customTypes *CustomTypeRegistry) []string {
	// Get all formats used in this DTO
	usedFormats := getUsedFormatsInDTO(dto)

	// Use the custom type registry to get the appropriate imports
	return customTypes.GetAllImports(usedFormats)
}

// getUsedFormatsInDTO finds all formats used in a single DTO
func getUsedFormatsInDTO(dto generator.DTO) []string {
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

// getPackageName returns the package name from config or default
func getPackageName(config generator.Config) string {
	if config.PackageName != "" {
		return config.PackageName
	}
	return "generated-schemas"
}
