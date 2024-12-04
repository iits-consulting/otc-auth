package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/golang/glog"
)

const stacktraceLevel = 2

func RemoveFromSliceAtIndex[T any](s []T, index int) []T {
	s[index] = s[len(s)-1]
	return s[:len(s)-1]
}

func WriteStringToFile(filepath string, content string) {
	outputFile, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
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
	if glog.V(stacktraceLevel) {
		// Also print stacktrace if verbosity is higher than 1
		glog.Fatal(err)
	} else {
		glog.Error(err)
		glog.Flush()
		os.Exit(2)
	}
}
