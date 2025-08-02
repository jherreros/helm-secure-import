package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractImages(t *testing.T) {
	testCases := []struct {
		name           string
		yamlInput      string
		expectedImages []string
		expectErr      bool
	}{
		{
			name: "Simple YAML with one image",
			yamlInput: `
image:
  repository: my-repo/my-image
  tag: 1.2.3
  pullPolicy: IfNotPresent
`,
			expectedImages: []string{"my-repo/my-image:1.2.3"},
		},
		{
			name: "Complex YAML with multiple images",
			yamlInput: `
image:
  repository: my-repo/my-image
  tag: 1.2.3
sidecar:
  image: my-repo/sidecar:latest
`,
			expectedImages: []string{"my-repo/my-image:1.2.3", "my-repo/sidecar:latest"},
		},
		{
			name: "YAML with no images",
			yamlInput: `
key: value
`,
			expectedImages: []string{},
		},
		{
			name: "YAML with invalid image format",
			yamlInput: `
image: invalid-image
`,
			expectedImages: []string{},
		},
		{
			name: "YAML with image in a sequence",
			yamlInput: `
images:
  - my-repo/image1:v1
  - my-repo/image2:v2
`,
			expectedImages: []string{"my-repo/image1:v1", "my-repo/image2:v2"},
		},
		{
			name: "YAML with duplicate images",
			yamlInput: `
image1:
  repository: my-repo/my-image
  tag: 1.2.3
image2:
  repository: my-repo/my-image
  tag: 1.2.3
`,
			expectedImages: []string{"my-repo/my-image:1.2.3"},
		},
		{
			name: "YAML with nested image structures",
			yamlInput: `
frontend:
  image:
    repository: my-repo/frontend
    tag: v1.0.0
backend:
  image:
    repository: my-repo/backend
    tag: v2.0.0
database:
  image: postgres:13
`,
			expectedImages: []string{"my-repo/backend:v2.0.0", "my-repo/frontend:v1.0.0", "postgres:13"},
		},
		{
			name: "YAML with registry prefix",
			yamlInput: `
image:
  repository: registry.example.com/my-repo/my-image
  tag: latest
`,
			expectedImages: []string{"registry.example.com/my-repo/my-image:latest"},
		},
		{
			name: "YAML with port in registry",
			yamlInput: `
image:
  repository: localhost:5000/my-image
  tag: dev
`,
			expectedImages: []string{"localhost:5000/my-image:dev"},
		},
		{
			name: "YAML with malformed repository-tag structure",
			yamlInput: `
image:
  repository: my-repo/my-image
  # Missing tag field
  pullPolicy: Always
`,
			expectedImages: []string{},
		},
		{
			name: "YAML with empty values",
			yamlInput: `
image:
  repository: ""
  tag: ""
another: my-repo/valid:v1
`,
			expectedImages: []string{"my-repo/valid:v1"},
		},
		{
			name: "YAML with semantic versioning",
			yamlInput: `
image:
  repository: my-repo/app
  tag: v1.2.3-alpha+build.123
`,
			expectedImages: []string{"my-repo/app:v1.2.3-alpha+build.123"},
		},
		{
			name: "YAML with mixed valid and invalid images",
			yamlInput: `
validImage: my-repo/valid:v1
invalidImage: "not-a-valid-image"
anotherValid:
  repository: registry.io/app
  tag: latest
`,
			expectedImages: []string{"my-repo/valid:v1", "registry.io/app:latest"},
		},
		{
			name: "YAML with whitespace in image names",
			yamlInput: `
image: "  my-repo/app:v1  "
`,
			expectedImages: []string{"my-repo/app:v1"},
		},
		{
			name: "Multiple YAML documents",
			yamlInput: `---
image:
  repository: my-repo/app1
  tag: v1
---
image:
  repository: my-repo/app2
  tag: v2
`,
			expectedImages: []string{"my-repo/app1:v1", "my-repo/app2:v2"},
		},
		{
			name: "Invalid YAML",
			yamlInput: `
image:
  repository: my-repo/app
  tag: v1
  invalid: [
`,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			images, err := extractImages([]byte(tc.yamlInput))

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tc.expectedImages, images)
			}
		})
	}
}
