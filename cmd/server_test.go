package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseUpstreams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
		err      bool
	}{
		{
			name:     "single address with port",
			input:    "8.8.8.8:53",
			expected: []string{"8.8.8.8:53"},
			err:      false,
		},
		{
			name:     "single address without port",
			input:    "8.8.8.8",
			expected: []string{"8.8.8.8:53"},
			err:      false,
		},
		{
			name:     "multiple addresses with and without ports",
			input:    "1.3.5.1:53,8.8.8.8",
			expected: []string{"1.3.5.1:53", "8.8.8.8:53"},
			err:      false,
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
			err:      false,
		},
		{
			name:     "invalid address",
			input:    "invalid_address",
			expected: []string{},
			err:      true,
		},
		{
			name:     "invalid address with valid address",
			input:    "invalid_address,8.8.8.8",
			expected: []string{"8.8.8.8:53"},
			err:      true,
		},
		{
			name:     "same addresses",
			input:    "8.8.8.8,8.8.8.8,8.8.8.8",
			expected: []string{"8.8.8.8:53"},
			err:      false,
		},
		{
			name:     "same addresses with different ports",
			input:    "8.8.8.8:53,8.8.8.8:54,8.8.8.8:55",
			expected: []string{"8.8.8.8:53", "8.8.8.8:54", "8.8.8.8:55"},
			err:      false,
		},
		{
			name:     "with spaces",
			input:    "8.8.8.8, 1.1.1.1",
			expected: []string{"8.8.8.8:53", "1.1.1.1:53"},
			err:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseUpstreams(tt.input)
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
