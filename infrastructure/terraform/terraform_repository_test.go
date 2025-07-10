package terraform_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"driftdetector/domain/models"
	tfrepo "driftdetector/infrastructure/terraform"
)

// MockStateParser is a mock implementation of the StateParser interface
type MockStateParser struct {
	ParseStateFunc func(ctx context.Context, path string) (*models.TerraformState, error)
}

func (m *MockStateParser) ParseState(ctx context.Context, path string) (*models.TerraformState, error) {
	if m.ParseStateFunc != nil {
		return m.ParseStateFunc(ctx, path)
	}
	return nil, nil
}

func TestNewTerraformRepository(t *testing.T) {
	t.Run("with custom parser", func(t *testing.T) {
		// Given
		mockParser := &MockStateParser{}

		// When
		repo := tfrepo.NewTerraformRepository(mockParser)

		// Then
		assert.NotNil(t, repo, "Repository should not be nil")
	})

	t.Run("with nil parser", func(t *testing.T) {
		// When
		repo := tfrepo.NewTerraformRepository(nil)

		// Then
		assert.NotNil(t, repo, "Repository should not be nil")
	})
}

func TestTerraformRepository_GetInstanceConfigs(t *testing.T) {
	t.Run("successful state parsing", func(t *testing.T) {
		// Given
		expectedState := &models.TerraformState{
			Version:         4,
			TerraformVersion: "1.0.0",
			Resources:       []models.TerraformResource{},
		}

		mockParser := &MockStateParser{
			ParseStateFunc: func(_ context.Context, path string) (*models.TerraformState, error) {
				if path == "test.tfstate" {
					return expectedState, nil
				}
				return nil, nil
			},
		}

		repo := tfrepo.NewTerraformRepository(mockParser)

		// When
		instances, err := repo.GetInstanceConfigs(context.Background(), "test.tfstate")

		// Then
		assert.NoError(t, err, "Should not return an error")
		assert.NotNil(t, instances, "Should return instances slice")
	})

	t.Run("error from parser", func(t *testing.T) {
		// Given
		expectedErr := assert.AnError

		mockParser := &MockStateParser{
			ParseStateFunc: func(_ context.Context, path string) (*models.TerraformState, error) {
				if path == "invalid.tfstate" {
					return nil, expectedErr
				}
				return nil, nil
			},
		}

		repo := tfrepo.NewTerraformRepository(mockParser)

		// When
		instances, err := repo.GetInstanceConfigs(context.Background(), "invalid.tfstate")

		// Then
		assert.ErrorIs(t, err, expectedErr, "Should return the expected error")
		assert.Nil(t, instances, "Should not return any instances on error")
	})
}

func TestTerraformRepository_GetInstanceConfigsFromDir(t *testing.T) {
	t.Run("valid directory with state files", func(t *testing.T) {
		// Given
		tempDir, err := os.MkdirTemp("", "terraform-test-*")
		require.NoError(t, err, "Failed to create temp dir")
		defer os.RemoveAll(tempDir)

		// Create test files
		validState := []byte(`{"version": 4, "terraform_version": "1.0.0", "resources": []}`)
		validFile := filepath.Join(tempDir, "valid.tfstate")
		err = os.WriteFile(validFile, validState, 0644)
		require.NoError(t, err, "Failed to write test file")

		mockParser := &MockStateParser{
			ParseStateFunc: func(_ context.Context, path string) (*models.TerraformState, error) {
				return &models.TerraformState{
					Version:         4,
					TerraformVersion: "1.0.0",
					Resources:       []models.TerraformResource{},
				}, nil
			},
		}

		repo := tfrepo.NewTerraformRepository(mockParser)

		// When
		instances, err := repo.GetInstanceConfigsFromDir(context.Background(), tempDir)

		// Then
		assert.NoError(t, err, "Should not return an error")
		assert.NotNil(t, instances, "Should return instances slice")
		// Note: The actual extraction of instances is not implemented yet, so we can't test the count
	})

	t.Run("non-existent directory", func(t *testing.T) {
		// Given
		repo := tfrepo.NewTerraformRepository(&tfrepo.StateFileParser{})

		// When
		instances, err := repo.GetInstanceConfigsFromDir(context.Background(), "/non/existent/directory")

		// Then
		assert.Error(t, err, "Should return an error for non-existent directory")
		assert.Nil(t, instances, "Should not return any instances on error")
	})

	t.Run("directory with invalid state files", func(t *testing.T) {
		// Given
		tempDir, err := os.MkdirTemp("", "terraform-test-*")
		require.NoError(t, err, "Failed to create temp dir")
		defer os.RemoveAll(tempDir)

		// Create invalid state file
		invalidState := []byte(`invalid json`)
		invalidFile := filepath.Join(tempDir, "invalid.tfstate")
		err = os.WriteFile(invalidFile, invalidState, 0644)
		require.NoError(t, err, "Failed to write test file")

		repo := tfrepo.NewTerraformRepository(&tfrepo.StateFileParser{})

		// When
		instances, err := repo.GetInstanceConfigsFromDir(context.Background(), tempDir)

		// Then
		assert.NoError(t, err, "Should not return an error for invalid files")
		assert.NotNil(t, instances, "Should return instances slice")
		assert.Empty(t, instances, "Should return empty slice for invalid files")
	})
}
