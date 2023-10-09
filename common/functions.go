package common

import (
	"bytes"
	"encoding/json"
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
		log.Fatalf("fatal: error creating output file.\ntrace: %s", err)
	}

	_, err = outputFile.WriteString(content)
	if err != nil {
		log.Fatalf("fatal: error writing to file.\ntrace: %s", err)
	}
	err = outputFile.Close()
	if err != nil {
		log.Fatalf("fatal: error closing file.\ntrace: %s", err)
	}
}

func ByteSliceToIndentedJSONFormat(biteSlice []byte) string {
	var formattedJSON bytes.Buffer
	err := json.Indent(&formattedJSON, biteSlice, "", "   ")
	if err != nil {
		log.Fatal(err)
	}
	return formattedJSON.String()
}

func DeserializeJSONForType[T any](data []byte) *T {
	var pointer T
	err := json.Unmarshal(data, &pointer)
	if err != nil {
		log.Fatalf("fatal: error deserializing json.\ntrace: %s", err)
	}

	return &pointer
}
