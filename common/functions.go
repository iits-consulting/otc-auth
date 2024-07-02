package common

import (
	"bytes"
	"encoding/json"
	"fmt"
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
		ThrowError(fmt.Errorf("fatal: error creating output file.\ntrace: %w", err))
	}

	_, err = outputFile.WriteString(content)
	if err != nil {
		ThrowError(fmt.Errorf("fatal: error writing to file.\ntrace: %w", err))
	}
	err = outputFile.Close()
	if err != nil {
		ThrowError(fmt.Errorf("fatal: error closing file.\ntrace: %w", err))
	}
}

func ByteSliceToIndentedJSONFormat(biteSlice []byte) string {
	var formattedJSON bytes.Buffer
	err := json.Indent(&formattedJSON, biteSlice, "", "   ")
	if err != nil {
		ThrowError(err)
	}
	return formattedJSON.String()
}

func DeserializeJSONForType[T any](data []byte) *T {
	var pointer T
	err := json.Unmarshal(data, &pointer)
	if err != nil {
		ThrowError(fmt.Errorf("fatal: error deserializing json.\ntrace: %w", err))
	}

	return &pointer
}

func ThrowError(err error) {
	if glog.V(2) {
		// Also print stacktrace if verbosity is higher than 1
		glog.Fatal(err)
	} else {
		glog.Error(err)
		glog.Flush()
		//nolint:gomnd // glog.Fatal() also uses 2
		os.Exit(2)
	}
}
