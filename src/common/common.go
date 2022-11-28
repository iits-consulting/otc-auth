package common

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

func WriteStringToFile(filepath string, content string) {
	outputFile, err := os.Create(filepath)
	if err != nil {
		OutputErrorToConsoleAndExit(err, "fatal: error creating output file: %s")
	}

	_, err = outputFile.WriteString(content)
	if err != nil {
		OutputErrorToConsoleAndExit(err)
	}
	outputFile.Close()
}

func ReadFileContent(filepath string) (output string, err error) {
	if !fileExists(filepath) {
		return "", nil
	}
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	fileScanner := bufio.NewScanner(file)
	fileScanner.Scan()

	if err := fileScanner.Err(); err != nil {
		OutputErrorToConsoleAndExit(err, "fatal: error reading file content: %S")
	}

	return fileScanner.Text(), err
}

func GetHomeDir() (homeDir string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		OutputErrorToConsoleAndExit(err, "fatal: invalid home directory: %s")
	}

	return homeDir
}

func OutputErrorToConsoleAndExit(err error, errorMessage ...string) {
	if errorMessage != nil {
		fmt.Fprintf(os.Stderr, errorMessage[0], err)
	} else {
		fmt.Fprintf(os.Stderr, "fatal: %s", err)
	}

	os.Exit(1)
}

func OutputErrorMessageToConsoleAndExit(errorMessage string) {
	fmt.Println(errorMessage)
	os.Exit(1)
}

func ErrorMessageToIndentedJsonFormat(errorMessage []byte) string {
	var formattedJson bytes.Buffer
	err := json.Indent(&formattedJson, errorMessage, "", " ")
	if err != nil {
		OutputErrorToConsoleAndExit(err)
	}
	return formattedJson.String()
}
