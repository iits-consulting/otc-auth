package common

import (
	"net/http"
)

var (
	httpClientWithUnscopedToken = http.Client{Transport: roundTripHeaderTransport(nil)}
	httpClient                  = http.Client{}
)

func GetHttpClient() http.Client {
	defer httpClient.CloseIdleConnections()
	return httpClient
}

func GetHttpClientWithUnscopedToken() http.Client {
	defer httpClientWithUnscopedToken.CloseIdleConnections()
	return httpClientWithUnscopedToken
}

func (adt *RoundTripHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("X-Auth-Token", ReadOrCreateOTCAuthCredentialsFile().UnscopedToken.Value)
	return adt.T.RoundTrip(req)
}

func roundTripHeaderTransport(T http.RoundTripper) *RoundTripHeaderTransport {
	if T == nil {
		T = http.DefaultTransport
	}
	return &RoundTripHeaderTransport{T}
}
