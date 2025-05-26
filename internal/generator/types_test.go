package generator

import (
	"testing"
)

func TestPrimitiveType_TypeName(t *testing.T) {
	tests := []struct {
		name     string
		primType PrimitiveType
		expected string
	}{
		{"String type", PrimitiveType{Name: "string"}, "string"},
		{"Number type", PrimitiveType{Name: "number"}, "number"},
		{"Boolean type", PrimitiveType{Name: "boolean"}, "boolean"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.primType.TypeName(); got != tt.expected {
				t.Errorf("TypeName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPrimitiveType_GetFormat(t *testing.T) {
	tests := []struct {
		name     string
		primType PrimitiveType
		expected string
	}{
		{"No format", PrimitiveType{Name: "string"}, ""},
		{"Date-time format", PrimitiveType{Name: "string", Format: "date-time"}, "date-time"},
		{"UUID format", PrimitiveType{Name: "string", Format: "uuid"}, "uuid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.primType.GetFormat(); got != tt.expected {
				t.Errorf("GetFormat() = %v, want %v", got, tt.expected)
			}
		})
	}
}
