package iam

import (
	"net/http"
	"otc-cli/util"
)

type AddHeaderTransport struct {
	T http.RoundTripper
}

func (adt *AddHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("X-Auth-Token", util.ReadOrCreateOTCInfoFromFile().UnscopedToken.Value)
	return adt.T.RoundTrip(req)
}

func newAddHeaderTransport(T http.RoundTripper) *AddHeaderTransport {
	if T == nil {
		T = http.DefaultTransport
	}
	return &AddHeaderTransport{T}
}

var httpClientWithUnscopedToken = http.Client{Transport: newAddHeaderTransport(nil)}

func GetHttpClientWithUnscopedToken() http.Client {
	defer httpClientWithUnscopedToken.CloseIdleConnections()
	return httpClientWithUnscopedToken
}

var httpClient = http.Client{}

func GetHttpClient() http.Client {
	defer httpClient.CloseIdleConnections()
	return httpClient
}
