package common

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
)

func HTTPClientMakeRequest(request *http.Request, skipTLS bool) *http.Response {
	tr := &http.Transport{
		//nolint:gosec // Needs to be explicitly set to true via a flag to skip TLS verification.
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipTLS},
	}
	httpClient := http.Client{Transport: tr}
	response, err := httpClient.Do(request)
	if err != nil {
		ThrowError(fmt.Errorf("fatal: error making a request %w", err))
	}

	defer httpClient.CloseIdleConnections()
	return response
}

func GetRequest(method string, url string, body io.Reader) *http.Request {
	request, err := http.NewRequestWithContext(context.Background(), method, url, body)
	if err != nil {
		ThrowError(fmt.Errorf(
			"fatal: error building %s request for url %s\ntrace: %w",
			method, url, err))
	}

	return request
}

func closeStreamCheckErr(body io.ReadCloser, err error) {
	errBodyClose := body.Close()
	if errBodyClose != nil {
		err = fmt.Errorf("fatal: %w\nfatal: error closing response body\n%w", err, errBodyClose)
	}
	if err != nil {
		ThrowError(err)
	}
}

func GetBodyBytesFromResponse(response *http.Response) []byte {
	var err error

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("fatal: error reading response body\n%w", err)
		closeStreamCheckErr(response.Body, err)
	}

	statusCodeStartsWith2 := regexp.MustCompile(`2\d{2}`)
	if !statusCodeStartsWith2.MatchString(strconv.Itoa(response.StatusCode)) {
		err = fmt.Errorf("fatal: status %s, body:\n%s", response.Status, bodyBytes)
		closeStreamCheckErr(response.Body, err)
	}

	defer closeStreamCheckErr(response.Body, err)
	return bodyBytes
}
