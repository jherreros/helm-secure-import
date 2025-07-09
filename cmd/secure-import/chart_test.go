package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractImages(t *testing.T) {
	testCases := []struct {
		name          string
		yamlInput     string
		expectedImages []string
		expectErr     bool
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
