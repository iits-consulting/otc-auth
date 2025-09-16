package common

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type HTTPClient interface {
	MakeRequest(request *http.Request) (*http.Response, error)
}

type HTTPClientImpl struct {
	client *http.Client
}

func NewHTTPClient(skipTLS bool) HTTPClient {
	tr := &http.Transport{
		//nolint:gosec // Needs to be explicitly set to true via a flag to skip TLS verification.
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipTLS},
	}

	client := &http.Client{
		Transport: tr,
	}

	return &HTTPClientImpl{client: client}
}

func (c HTTPClientImpl) MakeRequest(request *http.Request) (*http.Response, error) {
	defer c.client.CloseIdleConnections()
	return c.client.Do(request)
}

func NewRequest(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf(
			"fatal: error building %s request for url %s\ntrace: %w",
			method, url, err)
	}

	return request, nil
}

func GetBodyBytesFromResponse(response *http.Response) ([]byte, error) {
	var err error

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("fatal: error reading response body\n%w", err)
		closeErr := response.Body.Close()
		return nil, errors.Join(err, closeErr)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		err = fmt.Errorf("fatal: status %s, body:\n%s", response.Status, bodyBytes)
		closeErr := response.Body.Close()
		return nil, errors.Join(err, closeErr)
	}

	return bodyBytes, response.Body.Close()
}
