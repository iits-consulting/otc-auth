package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
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
		_, errPrint := fmt.Fprintf(os.Stderr, errorMessage[0], err)
		if errPrint != nil {
			OutputErrorToConsoleAndExit(errPrint)
		}
	} else {
		_, errPrint := fmt.Fprintf(os.Stderr, "fatal: %s", err)
		if errPrint != nil {
			OutputErrorToConsoleAndExit(errPrint)
		}
	}

	os.Exit(1)
}

func OutputErrorMessageToConsoleAndExit(errorMessage string) {
	log.Println(errorMessage)
	os.Exit(1)
}

func ByteSliceToIndentedJSONFormat(biteSlice []byte) string {
	var formattedJSON bytes.Buffer
	err := json.Indent(&formattedJSON, biteSlice, "", "   ")
	if err != nil {
		OutputErrorToConsoleAndExit(err)
	}
	return formattedJSON.String()
}

func DeserializeJSONForType[T any](data []byte) *T {
	var pointer T
	err := json.Unmarshal(data, &pointer)
	if err != nil {
		OutputErrorToConsoleAndExit(err, "fatal: error deserializing json.\ntrace: %s")
	}

	return &pointer
}
