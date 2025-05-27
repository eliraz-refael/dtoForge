package generator

import (
	"fmt"
	"strings"
)

// PrimitiveType represents basic types like string, number, etc.
type PrimitiveType struct {
	Name   string `json:"name"`
	Format string `json:"format,omitempty"` // date-time, uuid, email, etc.
}

func (p PrimitiveType) TypeName() string  { return p.Name }
func (p PrimitiveType) GetFormat() string { return p.Format } // Added this method

// ObjectType represents a nested object type.
type ObjectType struct {
	DTORef  *DTO   `json:"dtoRef,omitempty"`
	RefName string `json:"refName,omitempty"`
	Inline  bool   `json:"inline"`
}

func (o ObjectType) TypeName() string {
	if o.RefName != "" {
		return o.RefName
	}
	if o.DTORef != nil {
		return o.DTORef.Name
	}
	return "object"
}

// ArrayType represents an array of elements.
type ArrayType struct {
	ElementType IRType `json:"elementType"`
}

func (a ArrayType) TypeName() string {
	return fmt.Sprintf("Array<%s>", a.ElementType.TypeName())
}

// ReferenceType represents a reference to an already defined DTO.
type ReferenceType struct {
	RefName string `json:"refName"`
}

func (r ReferenceType) TypeName() string { return r.RefName }

// EnumType represents an enum type.
type EnumType struct {
	Name           string   `json:"name"`
	UnderlyingType string   `json:"underlyingType"`
	Values         []string `json:"values"`
}

func (e EnumType) TypeName() string { return e.Name }

// UnionType represents oneOf/anyOf schemas
type UnionType struct {
	Types []IRType `json:"types"`
}

func (u UnionType) TypeName() string {
	var typeNames []string
	for _, t := range u.Types {
		typeNames = append(typeNames, t.TypeName())
	}
	return fmt.Sprintf("(%s)", strings.Join(typeNames, " | "))
}
