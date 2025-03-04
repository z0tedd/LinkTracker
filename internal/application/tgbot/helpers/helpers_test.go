package helpers_test

import (
	"testing"

	"github.com/central-university-dev/go-z0tedd/internal/application/tgbot/helpers"
	"github.com/stretchr/testify/assert"
)

func TestIsSupported(t *testing.T) {
	// Define test cases with input URLs and their expected validity
	testCases := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "Valid GitHub URL without trailing slash",
			url:      "https://github.com/vektra/mockery",
			expected: true,
		},
		{
			name:     "Valid GitHub URL with additional path",
			url:      "https://github.com/user/repo/issues/123",
			expected: true,
		},
		{
			name:     "Valid Stack Overflow URL with additional path",
			url:      "https://stackoverflow.com/questions/123456/how-to-do-something",
			expected: true,
		},
		{
			name:     "Valid Stack Overflow URL with trailing slash",
			url:      "https://stackoverflow.com/questions/789012/",
			expected: true,
		},
		{
			name:     "Valid Stack Overflow URL without trailing slash",
			url:      "https://stackoverflow.com/questions/2145590",
			expected: true,
		},
		{
			name:     "Invalid URL with incorrect format",
			url:      "https://example.com/some/path",
			expected: false,
		},
		{
			name:     "Invalid GitHub URL missing repository name",
			url:      "https://github.com/user",
			expected: false,
		},
		{
			name:     "Empty URL",
			url:      "",
			expected: false,
		},
	}

	// Iterate through each test case
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function and assert the result
			result := helpers.IsSupported(tc.url)
			assert.Equal(t, tc.expected, result, "Test case failed: %s", tc.name)
		})
	}
}

func TestIsValidURL(t *testing.T) {
	// Define test cases with input URLs and their expected validity
	testCases := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "Valid URL with http",
			url:      "http://example.com",
			expected: true,
		},
		{
			name:     "Valid URL with https",
			url:      "https://example.com",
			expected: true,
		},
		{
			name:     "Invalid URL without protocol",
			url:      "example.com",
			expected: false,
		},
		{
			name:     "Invalid URL with ftp protocol",
			url:      "ftp://example.com",
			expected: false,
		},
		{
			name:     "Empty URL",
			url:      "",
			expected: false,
		},
		{
			name:     "URL with only http prefix",
			url:      "http://",
			expected: true,
		},
		{
			name:     "URL with only https prefix",
			url:      "https://",
			expected: true,
		},
		{
			name:     "URL with invalid characters before protocol",
			url:      "invalidhttp://example.com",
			expected: false,
		},
	}

	// Iterate through each test case
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function and assert the result
			result := helpers.IsValidURL(tc.url)
			assert.Equal(t, tc.expected, result, "Test case failed: %s", tc.name)
		})
	}
}
