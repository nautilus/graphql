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
	io.ReaderAt
	io.Seeker
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

// function extracts attached files and sets respective variables to null
func extractFiles(input *QueryInput) *UploadMap {
	uploadMap := &UploadMap{}

	for varName, value := range input.Variables {
		switch valueTyped := value.(type) {
		case Upload:
			uploadMap.Add(valueTyped, varName)
			input.Variables[varName] = nil
		case []Upload:
			for i, upload := range valueTyped {
				uploadMap.Add(upload, fmt.Sprintf("%s.%d", varName, i))
			}
			input.Variables[varName] = nil
		default:
			//noop
		}
	}
	return uploadMap
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
