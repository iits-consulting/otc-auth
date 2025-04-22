package common

import (
	"context"
	"crypto/tls"
	"errors"
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

func closeStream(body io.ReadCloser) error {
	errBodyClose := body.Close()
	return errBodyClose
}

func GetBodyBytesFromResponse(response *http.Response) ([]byte, error) {
	var err error

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("fatal: error reading response body\n%w", err)
		closeErr := closeStream(response.Body)
		return nil, errors.Join(err, closeErr)
	}

	statusCodeStartsWith2 := regexp.MustCompile(`2\d{2}`)
	if !statusCodeStartsWith2.MatchString(strconv.Itoa(response.StatusCode)) {
		err = fmt.Errorf("fatal: status %s, body:\n%s", response.Status, bodyBytes)
		closeErr := closeStream(response.Body)
		return nil, errors.Join(err, closeErr)
	}

	return bodyBytes, closeStream(response.Body)
}
