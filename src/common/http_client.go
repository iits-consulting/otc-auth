package common

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
)

func HttpClientMakeRequest(request *http.Request) *http.Response {
	httpClient := http.Client{}
	response, err := httpClient.Do(request)
	if err != nil {
		OutputErrorToConsoleAndExit(err, "fatal: error making a request %s")
	}

	defer httpClient.CloseIdleConnections()
	return response
}

func GetRequest(method string, url string, body io.Reader) *http.Request {
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		OutputErrorMessageToConsoleAndExit(fmt.Sprintf("fatal: error building %s request for url %s\ntrace: %s", method, url, err.Error()))
	}

	return request
}

func GetBodyBytesFromResponse(response *http.Response) []byte {
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			OutputErrorToConsoleAndExit(err, "fatal: error closing response body.\ntrace: %s")
		}
	}(response.Body)

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		OutputErrorToConsoleAndExit(err, "fatal: error reading response body.\ntrace: %s")
	}

	statusCodeStartsWith2 := regexp.MustCompile(`2\d{2}`)
	if !statusCodeStartsWith2.MatchString(strconv.Itoa(response.StatusCode)) {
		errorMessage := fmt.Sprintf("error: status %s, body:\n%s", response.Status, ByteSliceToIndentedJsonFormat(bodyBytes))
		OutputErrorMessageToConsoleAndExit(errorMessage)
	}

	return bodyBytes
}
