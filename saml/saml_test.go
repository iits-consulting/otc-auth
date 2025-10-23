//nolint:testpackage // whitebox testing
package saml

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/common/headervalues"
	header "otc-auth/common/xheaders"

	"github.com/go-http-utils/headers"
)

type mockHTTPClient struct {
	T *testing.T

	ResponseToReturn *http.Response
	ErrorToReturn    error

	ExpectedURL    string
	ExpectedMethod string
	ExpectedHeader http.Header
	ExpectedBody   []byte
}

type mockSequencedHTTPClient struct {
	T         *testing.T
	Responses []*http.Response // A queue of responses to return for each call.
	Errors    []error          // A parallel queue of errors to return.
	callIndex int              // Tracks which call we are on.
}

type mockCredentialParser struct {
	TokenToReturn *common.TokenResponse
	ErrorToReturn error
}

func (m *mockCredentialParser) Parse(resp *http.Response) (*common.TokenResponse, error) {
	return m.TokenToReturn, m.ErrorToReturn
}

func (m *mockSequencedHTTPClient) MakeRequest(req *http.Request) (*http.Response, error) {
	if m.callIndex >= len(m.Responses) || m.callIndex >= len(m.Errors) {
		m.T.Fatalf("MakeRequest called more times than expected. Got call #%d", m.callIndex+1)
		return nil, fmt.Errorf("unexpected call")
	}

	response := m.Responses[m.callIndex]
	err := m.Errors[m.callIndex]
	m.callIndex++
	return response, err
}

func (m *mockHTTPClient) MakeRequest(req *http.Request) (*http.Response, error) {
	if m.ExpectedMethod != "" && req.Method != m.ExpectedMethod {
		m.T.Errorf("MakeRequest() received method %q, want %q", req.Method, m.ExpectedMethod)
	}
	if m.ExpectedURL != "" && req.URL.String() != m.ExpectedURL {
		m.T.Errorf("MakeRequest() received URL %q, want %q", req.URL.String(), m.ExpectedURL)
	}
	if m.ExpectedHeader != nil {
		headerKey := headers.ContentType
		if got := req.Header.Get(headerKey); got != m.ExpectedHeader.Get(headerKey) {
			m.T.Errorf("MakeRequest() header %q = %q, want %q", headerKey, got, m.ExpectedHeader.Get(headerKey))
		}
	}
	if m.ExpectedBody != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		if !bytes.Equal(bodyBytes, m.ExpectedBody) {
			m.T.Errorf("MakeRequest() body = %q, want %q", string(bodyBytes), string(m.ExpectedBody))
		}
	}

	return m.ResponseToReturn, m.ErrorToReturn
}

func TestAuthenticator_validateAuthenticationWithServiceProvider(t *testing.T) {
	type fields struct {
		client common.HTTPClient
	}
	type args struct {
		ctx               context.Context
		assertionResult   common.SamlAssertionResponse
		responseBodyBytes []byte
	}
	ctx := context.Background()
	expectedURL := "https://example.com/assertion-consumer"
	requestBodyBytes := []byte("<saml>assertion</saml>")

	validAssertionResult := common.SamlAssertionResponse{
		Name: xml.Name{},
		Header: struct {
			Response struct {
				AssertionConsumerServiceURL string `xml:"AssertionConsumerServiceURL,attr"`
			} `xml:"Response"`
		}{
			Response: struct {
				AssertionConsumerServiceURL string `xml:"AssertionConsumerServiceURL,attr"`
			}{
				AssertionConsumerServiceURL: expectedURL,
			},
		},
	}

	successResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("success")),
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *http.Response
		wantErr bool
	}{
		{
			name: "Success - Happy Path",
			fields: fields{
				client: &mockHTTPClient{
					T:                t,
					ResponseToReturn: successResponse,
					ErrorToReturn:    nil,
					ExpectedURL:      expectedURL,
					ExpectedMethod:   http.MethodPost,
					ExpectedHeader:   http.Header{headers.ContentType: []string{headervalues.ApplicationPaos}},
					ExpectedBody:     requestBodyBytes,
				},
			},
			args: args{
				ctx:               ctx,
				assertionResult:   validAssertionResult,
				responseBodyBytes: requestBodyBytes,
			},
			want:    successResponse,
			wantErr: false,
		},
		{
			name: "Failure - Client returns an error",
			fields: fields{
				client: &mockHTTPClient{
					T:             t,
					ErrorToReturn: fmt.Errorf("simulated network error"),
				},
			},
			args: args{
				ctx:               ctx,
				assertionResult:   validAssertionResult,
				responseBodyBytes: requestBodyBytes,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Failure - NewRequest fails due to bad URL",
			fields: fields{
				client: &mockHTTPClient{T: t},
			},
			args: args{
				ctx: ctx,
				assertionResult: common.SamlAssertionResponse{
					Header: struct {
						Response struct {
							AssertionConsumerServiceURL string `xml:"AssertionConsumerServiceURL,attr"`
						} `xml:"Response"`
					}{
						Response: struct {
							AssertionConsumerServiceURL string `xml:"AssertionConsumerServiceURL,attr"`
						}{
							AssertionConsumerServiceURL: "::not a valid URL",
						},
					},
				},
				responseBodyBytes: requestBodyBytes,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Authenticator{
				client: tt.fields.client,
			}
			got, err := a.validateAuthenticationWithServiceProvider(tt.args.ctx,
				tt.args.assertionResult, tt.args.responseBodyBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAuthenticationWithServiceProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("validateAuthenticationWithServiceProvider() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthenticator_authenticateWithIdp(t *testing.T) {
	// --- Common test data ---
	ctx := context.Background()
	authParams := common.AuthInfo{
		IdpURL:   "https://idp.example.com/login",
		Username: "testuser",
		Password: "testpassword",
	}

	// Create the expected Basic Auth header value.
	auth := authParams.Username + ":" + authParams.Password
	expectedAuthHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))

	// This is the body of the *incoming* response, which will be the body of the *outgoing* request.
	samlRequestBody := "<saml>request</saml>"
	samlResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(samlRequestBody)),
	}

	// This is the body of the response we expect to get back from the IdP.
	expectedResponseBody := []byte("<saml>response</saml>")
	idpSuccessResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(expectedResponseBody)),
	}

	type fields struct {
		client common.HTTPClient
	}
	type args struct {
		ctx          context.Context
		params       common.AuthInfo
		samlResponse *http.Response
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Success - Happy Path",
			fields: fields{
				client: &mockHTTPClient{
					T:                t,
					ResponseToReturn: idpSuccessResponse,
					ExpectedURL:      authParams.IdpURL,
					ExpectedMethod:   http.MethodPost,
					ExpectedHeader: http.Header{
						headers.ContentType:   []string{headervalues.TextXML},
						headers.Authorization: []string{expectedAuthHeader},
					},
				},
			},
			args: args{
				ctx:          ctx,
				params:       authParams,
				samlResponse: samlResponse,
			},
			want:    expectedResponseBody,
			wantErr: false,
		},
		{
			name: "Failure - Client returns an error",
			fields: fields{
				client: &mockHTTPClient{
					T:             t,
					ErrorToReturn: fmt.Errorf("simulated network error"),
				},
			},
			args: args{
				ctx:          ctx,
				params:       authParams,
				samlResponse: samlResponse,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Failure - NewRequest fails due to bad URL",
			fields: fields{
				client: &mockHTTPClient{T: t}, // Client won't be called.
			},
			args: args{
				ctx: ctx,
				params: common.AuthInfo{
					IdpURL: "::not a valid url", // Invalid URL to make NewRequest fail.
				},
				samlResponse: samlResponse,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Authenticator{
				client: tt.fields.client,
			}
			got, err := a.authenticateWithIdp(tt.args.ctx, tt.args.params, tt.args.samlResponse)
			if (err != nil) != tt.wantErr {
				t.Errorf("authenticateWithIdp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("authenticateWithIdp() got = %s, want %s", string(got), string(tt.want))
			}
		})
	}
}

func TestAuthenticator_getServiceProviderInitiatedRequest(t *testing.T) {
	ctx := context.Background()
	authParams := common.AuthInfo{
		IdpName:      "my-idp",
		AuthProtocol: "saml",
		Region:       "eu-de",
	}

	expectedURL := endpoints.IdentityProviders(authParams.IdpName, string(authParams.AuthProtocol), authParams.Region)

	successResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("success")),
	}

	type fields struct {
		client common.HTTPClient
	}
	type args struct {
		ctx    context.Context
		params common.AuthInfo
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *http.Response
		wantErr bool
	}{
		{
			name: "Success - Happy Path",
			fields: fields{
				client: &mockHTTPClient{
					T:                t,
					ResponseToReturn: successResponse,
					ExpectedURL:      expectedURL,
					ExpectedMethod:   http.MethodGet,
					ExpectedHeader: http.Header{
						headers.Accept: []string{headervalues.ApplicationPaos},
						header.Paos:    []string{headervalues.Paos},
					},
				},
			},
			args: args{
				ctx:    ctx,
				params: authParams,
			},
			want:    successResponse,
			wantErr: false,
		},
		{
			name: "Failure - Client returns an error",
			fields: fields{
				client: &mockHTTPClient{
					T:             t,
					ErrorToReturn: fmt.Errorf("simulated network error"),
				},
			},
			args: args{
				ctx:    ctx,
				params: authParams,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Failure - NewRequest fails (e.g., bad region)",
			fields: fields{
				client: &mockHTTPClient{T: t},
			},
			args: args{
				ctx: ctx,
				params: common.AuthInfo{
					Region: " ",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Authenticator{
				client: tt.fields.client,
			}
			got, err := a.getServiceProviderInitiatedRequest(tt.args.ctx, tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("getServiceProviderInitiatedRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getServiceProviderInitiatedRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthenticator_Authenticate(t *testing.T) {
	ctx := context.Background()
	authInfo := common.AuthInfo{
		Region: "eu-de",
	}

	spSuccessResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("sp-response-body")),
	}

	samlXMLBody := `
		<Header>
			<Response AssertionConsumerServiceURL="https://validate.example.com"></Response>
		</Header>
	`
	idpSuccessResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(samlXMLBody)),
	}

	finalTokenBody := `{"token": {"id": "final-token"}}`
	finalSuccessResponse := &http.Response{
		StatusCode: http.StatusCreated,
		Header:     http.Header{"X-Subject-Token": []string{"final-token-id"}},
		Body:       io.NopCloser(strings.NewReader(finalTokenBody)),
	}

	expectedTokenResponse := &common.TokenResponse{}

	type fields struct {
		client common.HTTPClient
		parser CredentialParser
	}
	type args struct {
		ctx      context.Context
		authInfo common.AuthInfo
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func() // Optional setup, e.g., for mocking package-level funcs
		want    *common.TokenResponse
		wantErr bool
	}{
		{
			name: "Success - Happy Path",
			fields: fields{
				client: &mockSequencedHTTPClient{
					T:         t,
					Responses: []*http.Response{spSuccessResponse, idpSuccessResponse, finalSuccessResponse},
					Errors:    []error{nil, nil, nil},
				},
				parser: &mockCredentialParser{TokenToReturn: expectedTokenResponse},
			},
			args:    args{ctx: ctx, authInfo: authInfo},
			want:    expectedTokenResponse,
			wantErr: false,
		},
		{
			name: "Failure - Final credential parsing fails",
			fields: fields{
				client: &mockSequencedHTTPClient{
					T:         t,
					Responses: []*http.Response{spSuccessResponse, idpSuccessResponse, finalSuccessResponse},
					Errors:    []error{nil, nil, nil},
				},
				// Configure the mock parser to return an error.
				parser: &mockCredentialParser{ErrorToReturn: fmt.Errorf("could not parse final token")},
			},
			args:    args{ctx: ctx, authInfo: authInfo},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Failure - First step (getServiceProviderInitiatedRequest) fails",
			fields: fields{
				client: &mockSequencedHTTPClient{
					T:         t,
					Responses: []*http.Response{nil},
					Errors:    []error{fmt.Errorf("network error on step 1")},
				},
			},
			args:    args{ctx: ctx, authInfo: authInfo},
			setup:   func() {},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Failure - Second step (authenticateWithIdp) fails",
			fields: fields{
				client: &mockSequencedHTTPClient{
					T:         t,
					Responses: []*http.Response{spSuccessResponse, nil},
					Errors:    []error{nil, fmt.Errorf("network error on step 2")},
				},
			},
			args:    args{ctx: ctx, authInfo: authInfo},
			setup:   func() {},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Failure - XML Unmarshal fails",
			fields: fields{
				client: &mockSequencedHTTPClient{
					T: t,
					Responses: []*http.Response{
						spSuccessResponse,
						{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("not valid xml"))},
					},
					Errors: []error{nil, nil},
				},
			},
			args:    args{ctx: ctx, authInfo: authInfo},
			setup:   func() {},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			a := &Authenticator{
				client: tt.fields.client,
				parser: tt.fields.parser,
			}
			got, err := a.Authenticate(tt.args.ctx, tt.args.authInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthenticateAndGetUnscopedToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AuthenticateAndGetUnscopedToken() got = %v, want %v", got, tt.want)
			}
		})
	}
}
