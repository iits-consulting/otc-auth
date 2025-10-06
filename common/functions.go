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

func ByteSliceToIndentedJSONFormat(biteSlice []byte) (string, error) {
	var formattedJSON bytes.Buffer
	err := json.Indent(&formattedJSON, biteSlice, "", "   ")
	if err != nil {
		return "", fmt.Errorf("couldn't indent json: %w", err)
	}
	return formattedJSON.String(), nil
}

func DeserializeJSONForType[Type any](data []byte) (*Type, error) {
	var content Type
	err := json.Unmarshal(data, &content)
	if err != nil {
		return nil, fmt.Errorf("fatal: error deserializing json.\ntrace: %w", err)
	}

	return &content, nil
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
