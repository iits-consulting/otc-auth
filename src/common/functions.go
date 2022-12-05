package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

func RemoveFromSliceAtIndex[T any](s []T, index int) []T {
	s[index] = s[len(s)-1]
	return s[:len(s)-1]
}

func WriteStringToFile(filepath string, content string) {
	outputFile, err := os.Create(filepath)
	if err != nil {
		OutputErrorToConsoleAndExit(err, "fatal: error creating output file.\ntrace: %s")
	}

	_, err = outputFile.WriteString(content)
	if err != nil {
		OutputErrorToConsoleAndExit(err, "fatal: error writing to file.\ntrace: %s")
	}
	err = outputFile.Close()
	if err != nil {
		OutputErrorToConsoleAndExit(err, "fatal: error closing file.\ntrace: %s")
	}
}

func OutputErrorToConsoleAndExit(err error, errorMessage ...string) {
	if errorMessage != nil {
		_, err := fmt.Fprintf(os.Stderr, errorMessage[0], err)
		if err != nil {
			OutputErrorToConsoleAndExit(err)
		}
	} else {
		_, err := fmt.Fprintf(os.Stderr, "fatal: %s", err)
		if err != nil {
			OutputErrorToConsoleAndExit(err)
		}
	}

	os.Exit(1)
}

func OutputErrorMessageToConsoleAndExit(errorMessage string) {
	fmt.Println(errorMessage)
	os.Exit(1)
}

func ByteSliceToIndentedJsonFormat(biteSlice []byte) string {
	var formattedJson bytes.Buffer
	err := json.Indent(&formattedJson, biteSlice, "", "   ")
	if err != nil {
		OutputErrorToConsoleAndExit(err)
	}
	return formattedJson.String()
}

func DeserializeJsonForType[T any](data []byte) *T {
	var pointer T
	err := json.Unmarshal(data, &pointer)
	if err != nil {
		OutputErrorToConsoleAndExit(err, "fatal: error deserializing json.\ntrace: %s")
	}

	return &pointer
}
