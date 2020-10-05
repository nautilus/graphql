package graphql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"strings"
	"testing"
)

func TestExtractFiles(t *testing.T) {

	upload1 := Upload{nil, "file1"}
	upload2 := Upload{nil, "file2"}
	upload3 := Upload{nil, "file3"}

	input := &QueryInput{
		Variables: map[string]interface{}{
			"stringParam": "hello world",
			"listParam":   []interface{}{"one", "two"},
			"someFile":    upload1,
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
	assert.Equal(t, "hello world", input.Variables["stringParam"])
	assert.Equal(t, []interface{}{"one", "two"}, input.Variables["listParam"])
}

func TestPrepareMultipart(t *testing.T) {
	upload1 := Upload{ioutil.NopCloser(bytes.NewBufferString("File1Contents")), "file1"}
	upload2 := Upload{ioutil.NopCloser(bytes.NewBufferString("File2Contents")), "file2"}
	upload3 := Upload{ioutil.NopCloser(bytes.NewBufferString("File3Contents")), "file3"}

	uploadMap := &UploadMap{}
	uploadMap.Add(upload1, "someFile")
	uploadMap.Add(upload2, "allFiles.0")
	uploadMap.Add(upload3, "allFiles.1")

	payload, _ := json.Marshal(map[string]interface{}{
		"query": "mutation TestFileUpload($someFile: Upload!,$allFiles: [Upload!]!) {upload(file: $someFile) uploadMulti(files: $allFiles)}",
		"variables": map[string]interface{}{
			"someFile": nil,
			"allFiles": []interface{}{nil, nil},
		},
		"operationName": "TestFileUpload",
	})

	body, contentType, err := prepareMultipart(payload, uploadMap)

	headerParts := strings.Split(contentType, "; boundary=")
	rawBody := []string{
		"--%[1]s",
		"Content-Disposition: form-data; name=\"operations\"",
		"",
		"{\"operationName\":\"TestFileUpload\",\"query\":\"mutation TestFileUpload($someFile: Upload!,$allFiles: [Upload!]!) {upload(file: $someFile) uploadMulti(files: $allFiles)}\",\"variables\":{\"allFiles\":[null,null],\"someFile\":null}}",
		"--%[1]s",
		"Content-Disposition: form-data; name=\"map\"",
		"",
		"{\"0\":[\"variables.someFile\"],\"1\":[\"variables.allFiles.0\"],\"2\":[\"variables.allFiles.1\"]}\n",
		"--%[1]s",
		"Content-Disposition: form-data; name=\"0\"; filename=\"file1\"",
		"Content-Type: application/octet-stream",
		"",
		"File1Contents",
		"--%[1]s",
		"Content-Disposition: form-data; name=\"1\"; filename=\"file2\"",
		"Content-Type: application/octet-stream",
		"",
		"File2Contents",
		"--%[1]s",
		"Content-Disposition: form-data; name=\"2\"; filename=\"file3\"",
		"Content-Type: application/octet-stream",
		"",
		"File3Contents",
		"--%[1]s--",
		"",
	}

	expected := fmt.Sprintf(strings.Join(rawBody, "\r\n"), headerParts[1])

	assert.Equal(t, "multipart/form-data", headerParts[0])
	assert.Equal(t, expected, string(body))
	assert.Nil(t, err)
}
