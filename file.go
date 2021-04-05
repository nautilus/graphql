package graphql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"strconv"
)

type File interface {
	io.Reader
	io.Closer
}
type Upload struct {
	File     File
	FileName string
}

type UploadMap []struct {
	upload    Upload
	positions []string
}

func (u *UploadMap) UploadMap() map[string][]string {
	var result = make(map[string][]string)

	for idx, attachment := range *u {
		result[strconv.Itoa(idx)] = attachment.positions
	}

	return result
}

func (u *UploadMap) NotEmpty() bool {
	return len(*u) > 0
}

func (u *UploadMap) Add(upload Upload, varName string) {
	*u = append(*u, struct {
		upload    Upload
		positions []string
	}{
		upload,
		[]string{fmt.Sprintf("variables.%s", varName)},
	})
}

// returns a map of file names to paths.
// Used only in testing extractFiles
func (u *UploadMap) uploads() map[string]string {
	var result = make(map[string]string)

	for _, attachment := range *u {
		result[attachment.upload.FileName] = attachment.positions[0]
	}

	return result
}

// function extracts attached files and sets respective variables to null
func extractFiles(input *QueryInput) *UploadMap {
	uploadMap := &UploadMap{}
	for varName, value := range input.Variables {
		uploadMap.extract(value, varName)
		if _, ok := value.(Upload); ok { //If the value was an upload, set the respective QueryInput variable to null
			input.Variables[varName] = nil
		}
	}
	return uploadMap
}

func (u *UploadMap) extract(value interface{}, path string) {
	switch val := value.(type) {
	case Upload: // Upload found
		u.Add(val, path)
	case map[string]interface{}:
		for k, v := range val {
			u.extract(v, fmt.Sprintf("%s.%s", path, k))
			if _, ok := v.(Upload); ok { //If the value was an upload, set the respective QueryInput variable to null
				val[k] = nil
			}
		}
	case []interface{}:
		for i, v := range val {
			u.extract(v, fmt.Sprintf("%s.%d", path, i))
			if _, ok := v.(Upload); ok { //If the value was an upload, set the respective QueryInput variable to null
				val[i] = nil
			}
		}
	}
	return
}

func prepareMultipart(payload []byte, uploadMap *UploadMap) (body []byte, contentType string, err error) {
	var b = bytes.Buffer{}
	var fw io.Writer

	w := multipart.NewWriter(&b)

	fw, err = w.CreateFormField("operations")
	if err != nil {
		return
	}

	_, err = fw.Write(payload)
	if err != nil {
		return
	}

	fw, err = w.CreateFormField("map")
	if err != nil {
		return
	}

	err = json.NewEncoder(fw).Encode(uploadMap.UploadMap())
	if err != nil {
		return
	}

	for index, uploadVariable := range *uploadMap {
		fw, err := w.CreateFormFile(strconv.Itoa(index), uploadVariable.upload.FileName)
		if err != nil {
			return b.Bytes(), w.FormDataContentType(), err
		}

		_, err = io.Copy(fw, uploadVariable.upload.File)
		if err != nil {
			return b.Bytes(), w.FormDataContentType(), err
		}
	}

	err = w.Close()
	if err != nil {
		return
	}

	return b.Bytes(), w.FormDataContentType(), nil
}
