package graphql

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractFiles(t *testing.T) {

	upload1 := Upload{nil, "file1"}
	upload2 := Upload{nil, "file2"}
	upload3 := Upload{nil, "file3"}

	input := &QueryInput{
		Variables: map[string]interface{}{
			"stringParam": "hello world",
			"someFile": upload1,
			"allFiles": []interface{}{
				upload2,
				upload3,
			},
			"integerParam": 10,
		},
	}

	actual := extractFiles(input)

	expected := &UploadMap{}
	expected.Add(upload1, "someFile")
	expected.Add(upload2, "allFiles.0")
	expected.Add(upload3, "allFiles.1")

	assert.Equal(t, expected, actual)
}
