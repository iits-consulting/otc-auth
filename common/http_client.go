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

type HTTPClient interface {
	MakeRequest(request *http.Request, skipTLS bool) (*http.Response, error)
}

type HTTPClientImpl struct{}

func (c HTTPClientImpl) MakeRequest(request *http.Request, skipTLS bool) (*http.Response, error) {
	tr := &http.Transport{
		//nolint:gosec // Needs to be explicitly set to true via a flag to skip TLS verification.
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipTLS},
	}
	httpClient := http.Client{Transport: tr}
	defer httpClient.CloseIdleConnections()
	return httpClient.Do(request)
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

func GetBodyBytesFromResponse(response *http.Response) ([]byte, error) {
	var err error

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("fatal: error reading response body\n%w", err)
		closeErr := response.Body.Close()
		return nil, errors.Join(err, closeErr)
	}

	statusCodeStartsWith2 := regexp.MustCompile(`2\d{2}`)
	if !statusCodeStartsWith2.MatchString(strconv.Itoa(response.StatusCode)) {
		err = fmt.Errorf("fatal: status %s, body:\n%s", response.Status, bodyBytes)
		closeErr := response.Body.Close()
		return nil, errors.Join(err, closeErr)
	}

	return bodyBytes, response.Body.Close()
}
