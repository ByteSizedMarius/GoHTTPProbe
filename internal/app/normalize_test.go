package app

import (
	"reflect"
	"testing"
)

func TestNormalizeHeaderFlags(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Empty input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "Single header",
			input:    []string{"User-Agent: Test"},
			expected: []string{"User-Agent: Test"},
		},
		{
			name:     "Multiple headers",
			input:    []string{"User-Agent: Test", "Accept: application/json"},
			expected: []string{"User-Agent: Test", "Accept: application/json"},
		},
		{
			name:     "Comma-separated headers",
			input:    []string{"User-Agent: Test, Accept: application/json"},
			expected: []string{"User-Agent: Test", "Accept: application/json"},
		},
		{
			name:     "Comma-separated headers with whitespace",
			input:    []string{"User-Agent: Test,   Accept: application/json  , Content-Type: text/plain"},
			expected: []string{"User-Agent: Test", "Accept: application/json", "Content-Type: text/plain"},
		},
		{
			name:     "Mixed single and comma-separated",
			input:    []string{"User-Agent: Test", "Accept: application/json, Content-Type: text/plain"},
			expected: []string{"User-Agent: Test", "Accept: application/json", "Content-Type: text/plain"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeHeaderFlags(tc.input)
			
			// Compare lengths first since empty slices might not be exactly equal
			if len(result) != len(tc.expected) {
				t.Errorf("Expected length %d, got %d", len(tc.expected), len(result))
			}

			// Then compare contents for non-empty slices
			if len(result) > 0 && !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}