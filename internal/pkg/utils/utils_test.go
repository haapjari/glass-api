package utils_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/haapjari/glass-api/internal/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindFile(t *testing.T) {
	// Setup a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "testdir")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir) // Clean up after the test

	// Create a test file in the temporary directory
	testFilename := "testfile.txt"
	testFilePath := filepath.Join(tempDir, testFilename)
	err = os.WriteFile(testFilePath, []byte("test content"), 0644)
	require.NoError(t, err)

	// Define test cases
	tests := []struct {
		name        string
		path        string
		filename    string
		expected    string
		expectError bool
	}{
		{
			name:        "File found",
			path:        tempDir,
			filename:    testFilename,
			expected:    testFilePath,
			expectError: false,
		},
		{
			name:        "File not found",
			path:        tempDir,
			filename:    "nonexistent.txt",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := utils.FindFile(tt.path, tt.filename)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
