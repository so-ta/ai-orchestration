package blocks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLLoader_LoadAll(t *testing.T) {
	// Get the directory of this test file
	testDir := filepath.Join("yaml")

	loader := NewYAMLLoader(testDir)
	blocks, err := loader.LoadAll()
	require.NoError(t, err)

	// Should have loaded some blocks
	assert.NotEmpty(t, blocks, "Should have loaded some blocks from yaml directory")

	// Check that blocks are sorted by slug
	for i := 1; i < len(blocks); i++ {
		assert.True(t, blocks[i-1].Slug < blocks[i].Slug,
			"Blocks should be sorted by slug: %s should come before %s",
			blocks[i-1].Slug, blocks[i].Slug)
	}
}

func TestYAMLLoader_LoadHTTPBlock(t *testing.T) {
	testDir := filepath.Join("yaml")
	loader := NewYAMLLoader(testDir)
	blocks, err := loader.LoadAll()
	require.NoError(t, err)

	// Find HTTP block
	var httpBlock *SystemBlockDefinition
	for _, b := range blocks {
		if b.Slug == "http" {
			httpBlock = b
			break
		}
	}

	require.NotNil(t, httpBlock, "HTTP block should be loaded")
	assert.Equal(t, "HTTP Request", httpBlock.Name.EN)
	assert.Equal(t, domain.BlockCategoryApps, httpBlock.Category)
	assert.Equal(t, domain.BlockSubcategoryWeb, httpBlock.Subcategory)
	assert.Equal(t, 2, httpBlock.Version)
	assert.True(t, httpBlock.Enabled)
	assert.NotEmpty(t, httpBlock.Code)
	assert.NotEmpty(t, httpBlock.ConfigSchema.EN)
}

func TestYAMLLoader_LoadBlockWithInheritance(t *testing.T) {
	testDir := filepath.Join("yaml")
	loader := NewYAMLLoader(testDir)
	blocks, err := loader.LoadAll()
	require.NoError(t, err)

	// Find REST API block (inherits from HTTP)
	var restAPIBlock *SystemBlockDefinition
	for _, b := range blocks {
		if b.Slug == "rest-api" {
			restAPIBlock = b
			break
		}
	}

	require.NotNil(t, restAPIBlock, "REST API block should be loaded")
	assert.Equal(t, "http", restAPIBlock.ParentBlockSlug)
	assert.NotEmpty(t, restAPIBlock.PreProcess)
	assert.NotEmpty(t, restAPIBlock.PostProcess)
}

func TestYAMLLoader_LoadBlockWithDeclarativeConfig(t *testing.T) {
	testDir := filepath.Join("yaml")
	loader := NewYAMLLoader(testDir)
	blocks, err := loader.LoadAll()
	require.NoError(t, err)

	// Find GitHub Create Issue block (uses declarative request/response)
	var githubBlock *SystemBlockDefinition
	for _, b := range blocks {
		if b.Slug == "github_create_issue" {
			githubBlock = b
			break
		}
	}

	require.NotNil(t, githubBlock, "GitHub Create Issue block should be loaded")
	assert.Equal(t, "github-api", githubBlock.ParentBlockSlug)

	// Check declarative request config
	require.NotNil(t, githubBlock.Request, "Request config should be set")
	assert.Equal(t, "/repos/{{owner}}/{{repo}}/issues", githubBlock.Request.URL)
	assert.Equal(t, "POST", githubBlock.Request.Method)
	assert.NotNil(t, githubBlock.Request.Body)

	// Check declarative response config
	require.NotNil(t, githubBlock.Response, "Response config should be set")
	assert.Equal(t, []int{200, 201}, githubBlock.Response.SuccessStatus)
	assert.NotEmpty(t, githubBlock.Response.OutputMapping)
	assert.Equal(t, "body.id", githubBlock.Response.OutputMapping["id"])
}

func TestYAMLLoader_LoadBlockWithConfigDefaults(t *testing.T) {
	testDir := filepath.Join("yaml")
	loader := NewYAMLLoader(testDir)
	blocks, err := loader.LoadAll()
	require.NoError(t, err)

	// Find Bearer API block (has config defaults)
	var bearerBlock *SystemBlockDefinition
	for _, b := range blocks {
		if b.Slug == "bearer-api" {
			bearerBlock = b
			break
		}
	}

	require.NotNil(t, bearerBlock, "Bearer API block should be loaded")
	assert.NotNil(t, bearerBlock.ConfigDefaults)
}

func TestYAMLLoader_LoadBlockWithErrorCodes(t *testing.T) {
	testDir := filepath.Join("yaml")
	loader := NewYAMLLoader(testDir)
	blocks, err := loader.LoadAll()
	require.NoError(t, err)

	// Find HTTP block (has error codes)
	var httpBlock *SystemBlockDefinition
	for _, b := range blocks {
		if b.Slug == "http" {
			httpBlock = b
			break
		}
	}

	require.NotNil(t, httpBlock)
	require.NotEmpty(t, httpBlock.ErrorCodes)

	// Check first error code
	assert.Equal(t, "HTTP_001", httpBlock.ErrorCodes[0].Code)
	assert.Equal(t, "CONNECTION_ERROR", httpBlock.ErrorCodes[0].Name.EN)
	assert.True(t, httpBlock.ErrorCodes[0].Retryable)
}

func TestYAMLLoader_NonExistentDirectory(t *testing.T) {
	loader := NewYAMLLoader("/non/existent/directory")
	blocks, err := loader.LoadAll()

	// Should not error, just return empty list
	require.NoError(t, err)
	assert.Empty(t, blocks)
}

func TestYAMLLoader_InvalidYAML(t *testing.T) {
	// Create temp directory with invalid YAML
	tmpDir, err := os.MkdirTemp("", "yaml_loader_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Write invalid YAML
	invalidYAML := `
slug: test
version: invalid - not a number
`
	err = os.WriteFile(filepath.Join(tmpDir, "invalid.yaml"), []byte(invalidYAML), 0644)
	require.NoError(t, err)

	loader := NewYAMLLoader(tmpDir)
	_, err = loader.LoadAll()
	assert.Error(t, err)
}

func TestYAMLLoader_MultiDocumentFile(t *testing.T) {
	// Create temp directory with multi-document YAML
	tmpDir, err := os.MkdirTemp("", "yaml_loader_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Write multi-document YAML
	multiDoc := `
slug: block1
version: 1
name: Block 1
category: apps
enabled: true
---
slug: block2
version: 1
name: Block 2
category: apps
enabled: true
---
slug: block3
version: 1
name: Block 3
category: apps
enabled: true
`
	err = os.WriteFile(filepath.Join(tmpDir, "multi.yaml"), []byte(multiDoc), 0644)
	require.NoError(t, err)

	loader := NewYAMLLoader(tmpDir)
	blocks, err := loader.LoadAll()
	require.NoError(t, err)

	assert.Len(t, blocks, 3)

	// Check each block
	slugs := make(map[string]bool)
	for _, b := range blocks {
		slugs[b.Slug] = true
	}
	assert.True(t, slugs["block1"])
	assert.True(t, slugs["block2"])
	assert.True(t, slugs["block3"])
}

func TestYAMLLoader_SlackBlock(t *testing.T) {
	testDir := filepath.Join("yaml")
	loader := NewYAMLLoader(testDir)
	blocks, err := loader.LoadAll()
	require.NoError(t, err)

	// Find Slack block
	var slackBlock *SystemBlockDefinition
	for _, b := range blocks {
		if b.Slug == "slack" {
			slackBlock = b
			break
		}
	}

	require.NotNil(t, slackBlock, "Slack block should be loaded")
	assert.Equal(t, "Slack", slackBlock.Name.EN)
	assert.Equal(t, "webhook", slackBlock.ParentBlockSlug)
	assert.Equal(t, domain.BlockSubcategorySlack, slackBlock.Subcategory)

	// Check declarative config
	require.NotNil(t, slackBlock.Request)
	assert.Equal(t, "POST", slackBlock.Request.Method)
	assert.NotNil(t, slackBlock.Request.Body)

	require.NotNil(t, slackBlock.Response)
	assert.Contains(t, slackBlock.Response.SuccessStatus, 200)
}
