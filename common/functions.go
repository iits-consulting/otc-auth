package common

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/golang/glog"
)

func RemoveFromSliceAtIndex[T any](s []T, index int) []T {
	s[index] = s[len(s)-1]
	return s[:len(s)-1]
}

func WriteStringToFile(filepath string, content string) {
	outputFile, err := os.Create(filepath)
	if err != nil {
		glog.Fatalf("fatal: error creating output file.\ntrace: %s", err)
	}

	_, err = outputFile.WriteString(content)
	if err != nil {
		glog.Fatalf("fatal: error writing to file.\ntrace: %s", err)
	}
	err = outputFile.Close()
	if err != nil {
		glog.Fatalf("fatal: error closing file.\ntrace: %s", err)
	}
}

func ByteSliceToIndentedJSONFormat(biteSlice []byte) string {
	var formattedJSON bytes.Buffer
	err := json.Indent(&formattedJSON, biteSlice, "", "   ")
	if err != nil {
		glog.Fatal(err)
	}
	return formattedJSON.String()
}

func DeserializeJSONForType[T any](data []byte) *T {
	var pointer T
	err := json.Unmarshal(data, &pointer)
	if err != nil {
		glog.Fatalf("fatal: error deserializing json.\ntrace: %s", err)
	}

	return &pointer
}
